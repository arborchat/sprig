package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

	"git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// SettingsService allows querying, updating, and saving settings.
type SettingsService interface {
	NotificationsGloballyAllowed() bool
	SetNotificationsGloballyAllowed(bool)
	AcknowledgedNoticeVersion() int
	SetAcknowledgedNoticeVersion(version int)
	AddSubscription(id string)
	RemoveSubscription(id string)
	Subscriptions() []string
	Address() string
	SetAddress(string)
	BottomAppBar() bool
	SetBottomAppBar(bool)
	DockNavDrawer() bool
	SetDockNavDrawer(bool)
	DarkMode() bool
	SetDarkMode(bool)
	ActiveArborIdentityID() *fields.QualifiedHash
	Identity() (*forest.Identity, error)
	DataPath() string
	Persist() error
	CreateIdentity(name string) error
	Builder() (*forest.Builder, error)
	UseOrchardStore() bool
	SetUseOrchardStore(bool)
}

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

	// whether the user wants the navigation drawer to dock to the side of
	// the UI instead of appearing on top
	DockNavDrawer bool

	// whether the user wants to use the beta Orchard store for node storage.
	// Will become default in future release.
	OrchardStore bool

	Subscriptions []string
}

type settingsService struct {
	subscriptionLock sync.Mutex
	Settings
	dataDir string
	// state used for authoring messages
	activePrivKey *openpgp.Entity
	activeIdCache *forest.Identity
}

var _ SettingsService = &settingsService{}

func newSettingsService(stateDir string) (SettingsService, error) {
	s := &settingsService{
		dataDir: stateDir,
	}
	if err := s.Load(); err != nil {
		log.Printf("no loadable settings file found; defaults will be used: %v", err)
	}
	s.DiscoverIdentities()
	return s, nil
}

func (s *settingsService) Load() error {
	jsonSettings, err := ioutil.ReadFile(s.SettingsFile())
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	if err = json.Unmarshal(jsonSettings, &s.Settings); err != nil {
		return fmt.Errorf("couldn't parse json settings: %w", err)
	}
	return nil
}

func (s *settingsService) AddSubscription(id string) {
	s.subscriptionLock.Lock()
	defer s.subscriptionLock.Unlock()
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

func (s *settingsService) RemoveSubscription(id string) {
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

func (s *settingsService) Subscriptions() []string {
	s.subscriptionLock.Lock()
	defer s.subscriptionLock.Unlock()
	var out []string
	out = append(out, s.Settings.Subscriptions...)
	return out
}

func (s *settingsService) DockNavDrawer() bool {
	return s.Settings.DockNavDrawer
}

func (s *settingsService) SetDockNavDrawer(shouldDock bool) {
	s.Settings.DockNavDrawer = shouldDock
}

func (s *settingsService) AcknowledgedNoticeVersion() int {
	return s.Settings.AcknowledgedNoticeVersion
}

func (s *settingsService) SetAcknowledgedNoticeVersion(version int) {
	s.Settings.AcknowledgedNoticeVersion = version
}

func (s *settingsService) NotificationsGloballyAllowed() bool {
	return s.Settings.NotificationsEnabled == nil || *s.Settings.NotificationsEnabled
}

func (s *settingsService) SetNotificationsGloballyAllowed(allowed bool) {
	s.Settings.NotificationsEnabled = &allowed
}

func (s *settingsService) ActiveArborIdentityID() *fields.QualifiedHash {
	return s.Settings.ActiveIdentity
}

func (s *settingsService) Address() string {
	return s.Settings.Address
}

func (s *settingsService) SetAddress(addr string) {
	s.Settings.Address = addr
}

func (s *settingsService) DataPath() string {
	return filepath.Join(s.dataDir, "data")
}

func (s *settingsService) BottomAppBar() bool {
	return s.Settings.BottomAppBar
}

func (s *settingsService) SetBottomAppBar(bottom bool) {
	s.Settings.BottomAppBar = bottom
}

func (s *settingsService) DarkMode() bool {
	return s.Settings.DarkMode
}

func (s *settingsService) SetDarkMode(enabled bool) {
	s.Settings.DarkMode = enabled
}

func (s *settingsService) UseOrchardStore() bool {
	return s.Settings.OrchardStore
}

func (s *settingsService) SetUseOrchardStore(enabled bool) {
	s.Settings.OrchardStore = enabled
}

func (s *settingsService) SettingsFile() string {
	return filepath.Join(s.dataDir, "settings.json")
}

func (s *settingsService) KeysDir() string {
	return filepath.Join(s.dataDir, "keys")
}

func (s *settingsService) IdentitiesDir() string {
	return filepath.Join(s.dataDir, "identities")
}

func (s *settingsService) DiscoverIdentities() error {
	idsDir, err := os.Open(s.IdentitiesDir())
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

func (s *settingsService) Identity() (*forest.Identity, error) {
	if s.ActiveIdentity == nil {
		return nil, fmt.Errorf("no identity configured")
	}
	if s.activeIdCache != nil {
		return s.activeIdCache, nil
	}
	idData, err := ioutil.ReadFile(filepath.Join(s.IdentitiesDir(), s.ActiveIdentity.String()))
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

func (s *settingsService) Signer() (forest.Signer, error) {
	if s.ActiveIdentity == nil {
		return nil, fmt.Errorf("no identity configured, therefore no private key")
	}
	var privkey *openpgp.Entity
	if s.activePrivKey != nil {
		privkey = s.activePrivKey
	} else {
		keyfilePath := filepath.Join(s.KeysDir(), s.ActiveIdentity.String())
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

func (s *settingsService) Builder() (*forest.Builder, error) {
	id, err := s.Identity()
	if err != nil {
		return nil, err
	}
	signer, err := s.Signer()
	if err != nil {
		return nil, err
	}
	builder := forest.As(id, signer)
	return builder, nil
}

func (s *settingsService) CreateIdentity(name string) (err error) {
	keysDir := s.KeysDir()
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

	idsDir := s.IdentitiesDir()
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
	return s.Persist()
}

func (s *settingsService) Persist() error {
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		return fmt.Errorf("couldn't marshal settings as json: %w", err)
	}
	err = ioutil.WriteFile(s.SettingsFile(), data, 0770)
	if err != nil {
		return fmt.Errorf("couldn't save settings file: %w", err)
	}
	return nil
}
