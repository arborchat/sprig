// +build mage

package main

import (
	"archive/zip"
	"io/ioutil"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var LINUX_BIN = "sprig"
var LINUX_ARCHIVE = "sprig-linux.tar.xz"
var WINDOWS_BIN = "sprig.exe"
var WINDOWS_ARCHIVE = "sprig-windows.zip"
var FPNAME = "chat.arbor.Client.Sprig"
var FPCONFIG = FPNAME + ".yml"
var FPBUILD = "pakbuild"
var FPREPO = "/data/fp-repo"

var Aliases = map[string]interface{}{
	"c":   Clean,
	"l":   Linux,
	"w":   Windows,
	"fp":  Flatpak,
	"run": FlatpakRun,
}

func goFlags(platform string) string {
	return "-ldflags=-X=main.Version=" + embeddedVersion() + " " + platformFlags(platform)
}

func platformFlags(platform string) string {
	switch platform {
	case "windows":
		return "-ldflags=-H=windowsgui"
	default:
		return ""
	}
}

func embeddedVersion() string {
	gitVersion, err := sh.Output("git", "describe", "--tags", "--dirty", "--always")
	if err != nil {
		return "git"
	}
	return gitVersion
}

// Build all binary targets
func All() {
	mg.Deps(Linux, Windows)
}

// Build for specific platforms with a given binary name.
func BuildFor(platform, binary string) error {
	_, err := sh.Exec(map[string]string{"GOOS": platform, "GOFLAGS": goFlags(platform)},
		os.Stdout, os.Stderr, "go", "build", "-o", binary, ".")
	if err != nil {
		return err
	}
	return nil
}

// Build Linux
func LinuxBin() error {
	return BuildFor("linux", LINUX_BIN)
}

// Build Linux and archive/compress binary
func Linux() error {
	mg.Deps(LinuxBin)
	return sh.Run("tar", "-cJf", LINUX_ARCHIVE, LINUX_BIN, "desktop-assets", "install-linux.sh", "appicon.png", "LICENSE.txt")
}

// Build Windows
func WindowsBin() error {
	platform := "windows"
	_, err := sh.Exec(map[string]string{"GOFLAGS": goFlags(platform)},
		os.Stdout, os.Stderr, "go", "run", "gioui.org/cmd/gogio", "-x", "-target", "windows", "-o", WINDOWS_BIN, ".")
	if err != nil {
		return err
	}
	return nil
}

// Build Windows binary and zip it up
func Windows() error {
	mg.Deps(WindowsBin)
	file, err := os.Create(WINDOWS_ARCHIVE)
	if err != nil {
		return err
	}
	zipWriter := zip.NewWriter(file)
	f, err := zipWriter.Create(WINDOWS_BIN)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadFile(WINDOWS_BIN)
	if err != nil {
		return err
	}
	_, err = f.Write(body)
	if err != nil {
		return err
	}
	err = zipWriter.Close()
	if err != nil {
		return err
	}
	return nil
}

// Build flatpak
func Flatpak() error {
	mg.Deps(FlatpakInit)
	return sh.Run("flatpak-builder", "--user", "--force-clean", FPBUILD, FPCONFIG)
}

// Get a shell within flatpak
func FlatpakShell() error {
	mg.Deps(FlatpakInit)
	return sh.Run("flatpak-builder", "--user", "--run", FPBUILD, FPCONFIG, "sh")
}

// Install flatpak
func FlatpakInstall() error {
	mg.Deps(FlatpakInit)
	return sh.Run("flatpak-builder", "--user", "--install", "--force-clean", FPBUILD, FPCONFIG)
}

// Run flatpak
func FlatpakRun() error {
	return sh.Run("flatpak", "run", FPNAME)
}

// Flatpak into repo
func FlatpakRepo() error {
	return sh.Run("flatpak-builder", "--user", "--force-clean", "--repo="+FPREPO, FPBUILD, FPCONFIG)
}

// Enable repos if this is your first time running flatpak
func FlatpakInit() error {
	err := sh.RunV("flatpak", "remote-add", "--user", "--if-not-exists", "flathub", "https://flathub.org/repo/flathub.flatpakrepo")
	if err != nil {
		return err
	}
	err = sh.Run("flatpak", "install", "--user", "flathub", "org.freedesktop.Sdk/x86_64/19.08")
	if err != nil {
		return err
	}
	err = sh.Run("flatpak", "install", "--user", "flathub", "org.freedesktop.Platform/x86_64/19.08")
	if err != nil {
		return err
	}
	return nil
}

// Clean up
func Clean() error {
	return sh.Run("rm", "-rf", WINDOWS_ARCHIVE, WINDOWS_BIN, LINUX_ARCHIVE, LINUX_BIN, FPBUILD)
}
