.PHONY: android_install

SOURCE = main.go go.mod go.sum

sprig.apk: $(SOURCE)
	go run gioui.org/cmd/gogio -target android .

android_install: sprig.apk
	adb install sprig.apk
