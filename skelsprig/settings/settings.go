package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.sr.ht/~gioverse/skel/scheduler"
	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

type Settings struct {
	// relay address to connect to
	Address string

	// user's local identity ID
	ActiveIdentity *fields.QualifiedHash

	// the version of the disclaimer that the user has accepted
	AcknowledgedNoticeVersion int

	// whether notifications are accepted. The nil state indicates that
	// the user has not changed this value, and should be treated as true.
	// TODO(whereswaldon): find a backwards-compatible way to handle this
	// elegantly.
	NotificationsEnabled *bool

	// whether the user wants the app bar anchored at the bottom of the UI
	BottomAppBar bool

	DarkMode bool

	// whether the user wants to use the beta Orchard store for node storage.
	// Will become default in future release.
	OrchardStore bool

	Subscriptions []string
}

// UpdateEvent announces updated application settings.
type UpdateEvent struct {
	Settings
	Err error
}

// Event announces the current application settings.
type Event struct {
	Settings
}

// Request asks that the current application settings be announced over the
// bus as an event.
type Request struct{}

type Service struct {
	conn             scheduler.Connection
	subscriptionLock sync.Mutex
	Settings
	dataDir string
	// state used for authoring messages
	activePrivKey *openpgp.Entity
	activeIdCache *forest.Identity
}

func New(stateDir string, conn scheduler.Connection) (*Service, error) {
	s := &Service{
		conn:    conn,
		dataDir: stateDir,
	}
	if err := s.load(); err != nil {
		log.Printf("no loadable settings file found; defaults will be used: %v", err)
	}
	s.discoverIdentities()
	go s.run()
	return s, nil
}

func (s *Service) run() {
	for event := range s.conn.Output() {
		changed := true
		switch event := event.(type) {
		case AddSubscriptionRequest:
			s.addSubscription(event.CommunityID)
		case RemoveSubscriptionRequest:
			s.removeSubscription(event.CommunityID)
		case ConnectRequest:
			s.setAddress(event.Address)
		case NotificationRequest:
			s.setNotificationsGloballyAllowed(event.Enabled)
		case OrchardRequest:
			s.setUseOrchardStore(event.Enabled)
		case BottomBarRequest:
			s.setBottomAppBar(event.Enabled)
		case Request:
			changed = false
			time.Sleep(time.Second)
			s.conn.Message(Event{Settings: s.Settings})
		default:
			changed = false
		}
		if changed {
			err := s.persist()
			s.conn.Message(UpdateEvent{
				Settings: s.Settings,
				Err:      err,
			})
		}
	}
}

func (s *Service) load() error {
	jsonSettings, err := ioutil.ReadFile(s.settingsFile())
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	if err = json.Unmarshal(jsonSettings, &s.Settings); err != nil {
		return fmt.Errorf("couldn't parse json settings: %w", err)
	}
	return nil
}

type AddSubscriptionRequest struct {
	CommunityID string
}

func (s *Service) addSubscription(id string) {
	found := false
	for _, comm := range s.Settings.Subscriptions {
		if comm == id {
			found = true
			break
		}
	}
	if !found {
		s.Settings.Subscriptions = append(s.Settings.Subscriptions, id)
	}
}

type RemoveSubscriptionRequest struct {
	CommunityID string
}

func (s *Service) removeSubscription(id string) {
	s.subscriptionLock.Lock()
	defer s.subscriptionLock.Unlock()
	length := len(s.Settings.Subscriptions)
	for i, comm := range s.Settings.Subscriptions {
		if comm == id {
			s.Settings.Subscriptions = append(s.Settings.Subscriptions[:i], s.Settings.Subscriptions[i+1:length]...)
			return
		}
	}
}

func (s *Service) SetAcknowledgedNoticeVersion(version int) {
	s.Settings.AcknowledgedNoticeVersion = version
}

// NotificationRequest changes the user's notification preferences.
type NotificationRequest struct {
	Enabled bool
}

func (s *Service) setNotificationsGloballyAllowed(allowed bool) {
	s.Settings.NotificationsEnabled = &allowed
}

// ConnectRequest asks the client to connect to a relay on the
// specified address.
type ConnectRequest struct {
	Address string
}

func (s *Service) setAddress(addr string) {
	s.Settings.Address = addr
}

func (s *Service) dataPath() string {
	return filepath.Join(s.dataDir, "data")
}

// BottomBarRequest configures whether the app bar is shown on the
// bottom.
type BottomBarRequest struct {
	Enabled bool
}

func (s *Service) setBottomAppBar(bottom bool) {
	s.Settings.BottomAppBar = bottom
}

// DarkModeRequest configures whether the app is in dark mode.
type DarkModeRequest struct {
	Enabled bool
}

func (s *Service) setDarkMode(enabled bool) {
	s.Settings.DarkMode = enabled
}

// OrchardRequest changes whether backend support for the orchard
// node store is enable.
type OrchardRequest struct {
	Enabled bool
}

func (s *Service) setUseOrchardStore(enabled bool) {
	s.Settings.OrchardStore = enabled
}

func (s *Service) settingsFile() string {
	return filepath.Join(s.dataDir, "settings.json")
}

func (s *Service) keysDir() string {
	return filepath.Join(s.dataDir, "keys")
}

func (s *Service) identitiesDir() string {
	return filepath.Join(s.dataDir, "identities")
}

func (s *Service) discoverIdentities() error {
	idsDir, err := os.Open(s.identitiesDir())
	if err != nil {
		return fmt.Errorf("failed opening identities directory: %w", err)
	}
	names, err := idsDir.Readdirnames(0)
	if err != nil {
		return fmt.Errorf("failed listing identities directory: %w", err)
	}
	name := names[0]
	id := &fields.QualifiedHash{}
	err = id.UnmarshalText([]byte(name))
	if err != nil {
		return fmt.Errorf("failed unmarshalling name of first identity %s: %w", name, err)
	}
	s.ActiveIdentity = id
	return nil
}

func (s *Service) identity() (*forest.Identity, error) {
	if s.ActiveIdentity == nil {
		return nil, fmt.Errorf("no identity configured")
	}
	if s.activeIdCache != nil {
		return s.activeIdCache, nil
	}
	idData, err := ioutil.ReadFile(filepath.Join(s.identitiesDir(), s.ActiveIdentity.String()))
	if err != nil {
		return nil, fmt.Errorf("failed reading identity data: %w", err)
	}
	identity, err := forest.UnmarshalIdentity(idData)
	if err != nil {
		return nil, fmt.Errorf("failed decoding identity data: %w", err)
	}
	s.activeIdCache = identity
	return identity, nil
}

func (s *Service) signer() (forest.Signer, error) {
	if s.ActiveIdentity == nil {
		return nil, fmt.Errorf("no identity configured, therefore no private key")
	}
	var privkey *openpgp.Entity
	if s.activePrivKey != nil {
		privkey = s.activePrivKey
	} else {
		keyfilePath := filepath.Join(s.keysDir(), s.ActiveIdentity.String())
		keyfile, err := os.Open(keyfilePath)
		if err != nil {
			return nil, fmt.Errorf("unable to read key file: %w", err)
		}
		defer keyfile.Close()
		privkey, err = openpgp.ReadEntity(packet.NewReader(keyfile))
		if err != nil {
			return nil, fmt.Errorf("unable to decode key data: %w", err)
		}
		s.activePrivKey = privkey
	}
	signer, err := forest.NewNativeSigner(privkey)
	if err != nil {
		return nil, fmt.Errorf("couldn't wrap privkey in forest signer: %w", err)
	}
	return signer, nil
}

func (s *Service) builder() (*forest.Builder, error) {
	id, err := s.identity()
	if err != nil {
		return nil, err
	}
	signer, err := s.signer()
	if err != nil {
		return nil, err
	}
	builder := forest.As(id, signer)
	return builder, nil
}

func (s *Service) createIdentity(name string) (err error) {
	keysDir := s.keysDir()
	if err := os.MkdirAll(keysDir, 0770); err != nil {
		return fmt.Errorf("failed creating key storage directory: %w", err)
	}
	keypair, err := openpgp.NewEntity(name, "sprig-generated arbor identity", "", &packet.Config{})
	if err != nil {
		return fmt.Errorf("failed generating new keypair: %w", err)
	}
	signer, err := forest.NewNativeSigner(keypair)
	if err != nil {
		return fmt.Errorf("failed wrapping keypair into Signer: %w", err)
	}
	identity, err := forest.NewIdentity(signer, name, []byte{})
	if err != nil {
		return fmt.Errorf("failed generating arbor identity from signer: %w", err)
	}
	id := identity.ID()

	keyFilePath := filepath.Join(keysDir, id.String())
	keyFile, err := os.OpenFile(keyFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		return fmt.Errorf("failed creating key file: %w", err)
	}
	defer func() {
		if err != nil {
			if err = keyFile.Close(); err != nil {
				err = fmt.Errorf("failed closing key file: %w", err)
			}
		}
	}()
	if err := keypair.SerializePrivateWithoutSigning(keyFile, nil); err != nil {
		return fmt.Errorf("failed saving private key: %w", err)
	}

	idsDir := s.identitiesDir()
	if err := os.MkdirAll(idsDir, 0770); err != nil {
		return fmt.Errorf("failed creating identity storage directory: %w", err)
	}
	idFilePath := filepath.Join(idsDir, id.String())

	idFile, err := os.OpenFile(idFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0660)
	if err != nil {
		return fmt.Errorf("failed creating identity file: %w", err)
	}
	defer func() {
		if err != nil {
			if err = idFile.Close(); err != nil {
				err = fmt.Errorf("failed closing identity file: %w", err)
			}
		}
	}()
	binIdent, err := identity.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed serializing new identity: %w", err)
	}
	if _, err := idFile.Write(binIdent); err != nil {
		return fmt.Errorf("failed writing identity: %w", err)
	}

	s.ActiveIdentity = id
	s.activePrivKey = keypair
	return s.persist()
}

func (s *Service) persist() error {
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		return fmt.Errorf("couldn't marshal settings as json: %w", err)
	}
	err = ioutil.WriteFile(s.settingsFile(), data, 0770)
	if err != nil {
		return fmt.Errorf("couldn't save settings file: %w", err)
	}
	return nil
}
