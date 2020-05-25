.PHONY: android_install

SOURCE = $(shell find . -name '*\.go') go.mod go.sum
APPID = chat.arbor.sprig

sprig.apk: $(SOURCE)
	go run gioui.org/cmd/gogio -target android -appid $(APPID) .

android_install: sprig.apk
	adb install sprig.apk

logs:
	adb logcat -s -T1 $(APPID):\*
