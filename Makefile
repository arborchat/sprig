.PHONY: android_install logs windows linux macos android clean fp fp-install fp-repo fp-run

SOURCE = $(shell find . -name '*\.go') go.mod go.sum
APPID := chat.arbor.sprig.dev
ANDROID_CONFIG = $(HOME)/.android
KEYSTORE = $(ANDROID_CONFIG)/debug.keystore

ANDROID_APK = sprig.apk
ANDROID_SDK_ROOT := $(ANDROID_HOME)

WINDOWS_BIN = sprig.exe
WINDOWS_ARCHIVE = sprig-windows.zip

LINUX_BIN = sprig
LINUX_ARCHIVE = sprig-linux.tar.xz
LINUX_FILES = $(LINUX_BIN) ./desktop-assets ./install-linux.sh ./appicon.png ./LICENSE.txt

FPNAME = chat.arbor.Client.Sprig
FPCONFIG = $(FPNAME).yml
FPBUILD = pakbuild
FPREPO := /data/fp-repo

MACOS_BIN = sprig-mac
MACOS_APP = sprig.app
MACOS_ARCHIVE = sprig-macos.tar.gz

IOS_APP = sprig.ipa
IOS_VERSION := 0

android: $(ANDROID_APK)

$(ANDROID_APK): $(SOURCE) $(KEYSTORE)
	env ANDROID_SDK_ROOT=$(ANDROID_SDK_ROOT) go run gioui.org/cmd/gogio -x -target android -appid $(APPID) .

$(KEYSTORE):
	mkdir -p $(ANDROID_CONFIG)
	keytool -genkey -v -keystore $(ANDROID_CONFIG)/debug.keystore -alias androiddebugkey -storepass android -keypass android -keyalg RSA -validity 14000

windows: $(WINDOWS_ARCHIVE)

$(WINDOWS_ARCHIVE): $(WINDOWS_BIN)
	zip $(WINDOWS_ARCHIVE) $(WINDOWS_BIN)

$(WINDOWS_BIN): $(SOURCE)
	env GOOS=windows go build -o $(WINDOWS_BIN) .

linux: $(LINUX_ARCHIVE)

$(LINUX_ARCHIVE): $(LINUX_BIN)
	tar -cJf $(LINUX_ARCHIVE) $(LINUX_FILES)

$(LINUX_BIN): $(SOURCE)
	env GOOS=linux go build -o $(LINUX_BIN) .

macos: $(MACOS_ARCHIVE)

$(MACOS_ARCHIVE): $(MACOS_APP)
	tar czf $(MACOS_ARCHIVE) $(MACOS_APP)

$(MACOS_APP): $(MACOS_BIN) $(MACOS_APP).template
	rm -rf $(MACOS_APP)
	cp -rv $(MACOS_APP).template $(MACOS_APP)
	cp $(MACOS_BIN) $(MACOS_APP)/Contents/MacOS/$(MACOS_BIN)
	codesign -s - $(MACOS_APP)

$(MACOS_BIN): $(SOURCE)
	env GOOS=darwin CGO_CFLAGS=-mmacosx-version-min=10.14 \
	CGO_LDFLAGS=-mmacosx-version-min=10.14 \
	go build -o $(MACOS_BIN) -ldflags -v .

ios: $(IOS_APP)

$(IOS_APP): $(SOURCE)
	gogio -target ios -appid chat.arbor.sprig -version $(IOS_VERSION) .

android_install: $(ANDROID_APK)
	adb install $(ANDROID_APK)

logs:
	adb logcat -s -T1 $(APPID):\*

fp:
	flatpak-builder --force-clean $(FPBUILD) $(FPCONFIG)

fp-shell:
	flatpak-builder --run $(FPBUILD) $(FPCONFIG) sh

fp-install:
	flatpak-builder --user --install --force-clean $(FPBUILD) $(FPCONFIG)

fp-run:
	flatpak run $(FPCONFIG)

fp-repo:
	flatpak-builder --force-clean --repo=$(FPREPO) $(FPBUILD) $(FPCONFIG)

clean:
	rm -rf $(ANDROID_APK) $(WINDOWS_ARCHIVE) \
	    $(WINDOWS_BIN) $(LINUX_ARCHIVE) $(LINUX_BIN) \
	    $(MACOS_ARCHIVE) $(MACOS_BIN) $(FPBUILD)
