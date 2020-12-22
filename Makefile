.PHONY: android_install logs windows linux macos android clean fp fp-install fp-repo fp-run

SOURCE = $(shell find . -name '*\.go') go.mod go.sum
APPID := chat.arbor.sprig.dev
ANDROID_CONFIG = $(HOME)/.android
KEYSTORE = $(ANDROID_CONFIG)/debug.keystore

EMBEDDED_VERSION := $(shell git describe --tags --dirty --always || echo "git")

GOFLAGS := -ldflags=-X=main.Version="$(EMBEDDED_VERSION)"

ANDROID_APK = sprig.apk
ANDROID_SDK_ROOT := $(ANDROID_HOME)

MACOS_BIN = sprig-mac
MACOS_APP = sprig.app
MACOS_ARCHIVE = sprig-macos.tar.gz

IOS_APP = sprig.ipa
IOS_VERSION := 0

tag:
	echo "flags" $(GOFLAGS)

android: $(ANDROID_APK)

$(ANDROID_APK): $(SOURCE) $(KEYSTORE)
	env ANDROID_SDK_ROOT=$(ANDROID_SDK_ROOT) go run gioui.org/cmd/gogio $(GOFLAGS) -x -target android -appid $(APPID) .

$(KEYSTORE):
	mkdir -p $(ANDROID_CONFIG)
	keytool -genkey -v -keystore $(ANDROID_CONFIG)/debug.keystore -alias androiddebugkey -storepass android -keypass android -keyalg RSA -validity 14000

windows:
	mage windows

linux:
	mage linux

macos: $(MACOS_ARCHIVE)

$(MACOS_ARCHIVE): $(MACOS_APP)
	tar czf $(MACOS_ARCHIVE) $(MACOS_APP)

$(MACOS_APP): $(MACOS_BIN) $(MACOS_APP).template
	rm -rf $(MACOS_APP)
	cp -rv $(MACOS_APP).template $(MACOS_APP)
	mkdir -p $(MACOS_APP)/Contents/MacOS
	cp $(MACOS_BIN) $(MACOS_APP)/Contents/MacOS/$(MACOS_BIN)
	mkdir -p $(MACOS_APP)/Contents/Resources
	go install github.com/jackmordaunt/icns/cmd/icnsify && go mod tidy
	cat appicon.png | icnsify > $(MACOS_APP)/Contents/Resources/sprig.icns
	codesign -s - $(MACOS_APP)

$(MACOS_BIN): $(SOURCE)
	env GOOS=darwin GOFLAGS=$(GOFLAGS) CGO_CFLAGS=-mmacosx-version-min=10.14 \
	CGO_LDFLAGS=-mmacosx-version-min=10.14 \
	go build -o $(MACOS_BIN) -ldflags -v .

ios: $(IOS_APP)

$(IOS_APP): $(SOURCE)
	 go run gioui.org/cmd/gogio $(GOFLAGS) -target ios -appid chat.arbor.sprig -version $(IOS_VERSION) .

android_install: $(ANDROID_APK)
	adb install $(ANDROID_APK)

logs:
	adb logcat -s -T1 $(APPID):\*

fp:
	mage flatpak

fp-shell:
	mage flatpakShell

fp-install:
	mage flatpakInstall

fp-run:
	mage flatpakRun

fp-repo:
	mage flatpakRepo

clean:
	mage clean
