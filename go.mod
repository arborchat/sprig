module git.sr.ht/~whereswaldon/sprig

go 1.14

require (
	gioui.org v0.0.0-20210421151739-2296c80d288b
	gioui.org/cmd v0.0.0-20210422101526-9dae29844c9f
	gioui.org/x v0.0.0-20210605020051-c3156aa86f01
	gioui.org/x/haptic v0.0.0-20210120222453-b55819bc712b
	gioui.org/x/notify v0.0.0-20210117185607-25b1f7920092
	git.sr.ht/~athorp96/forest-ex v0.0.0-20210604181634-7063d1aadd25
	git.sr.ht/~whereswaldon/forest-go v0.0.0-20210610022432-d59588450728
	git.sr.ht/~whereswaldon/latest v0.0.0-20210304001450-aafd2a13a1bb
	git.sr.ht/~whereswaldon/sprout-go v0.0.0-20210408013049-fedf4ae2e7f8
	github.com/inkeliz/giohyperlink v0.0.0-20201127153708-cb2dff56ac99
	github.com/magefile/mage v1.10.0
	github.com/pkg/profile v1.5.0
	golang.org/x/crypto v0.0.0-20210415154028-4f45737414dc
	golang.org/x/exp v0.0.0-20201229011636-eab1b5eb1a03
	golang.org/x/image v0.0.0-20201208152932-35266b937fa6 // indirect
	golang.org/x/mod v0.4.0 // indirect
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/tools v0.0.0-20201222163215-f2e330f49058 // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200605105621-11f6ee2dd602

replace gioui.org/x => ../../gioui/x/
