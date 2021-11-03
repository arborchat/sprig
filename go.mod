module git.sr.ht/~whereswaldon/sprig

go 1.14

require (
	gioui.org v0.0.0-20211026101311-9cf7cc75f468
	gioui.org/cmd v0.0.0-20210804105805-1efe68c1540b
	gioui.org/x v0.0.0-20211102210401-cead9283b8ff
	gioui.org/x/haptic v0.0.0-20210120222453-b55819bc712b
	gioui.org/x/notify v0.0.0-20210117185607-25b1f7920092
	git.sr.ht/~athorp96/forest-ex v0.0.0-20210604181634-7063d1aadd25
	git.sr.ht/~gioverse/chat v0.0.0-20211102210743-a2a29f81c013
	git.sr.ht/~gioverse/skel v0.0.0-20211008142525-ecdaf33bb3a7
	git.sr.ht/~whereswaldon/forest-go v0.0.0-20210721201741-28efb6fd5020
	git.sr.ht/~whereswaldon/latest v0.0.0-20210304001450-aafd2a13a1bb
	git.sr.ht/~whereswaldon/sprout-go v0.0.0-20210408013049-fedf4ae2e7f8
	github.com/inkeliz/giohyperlink v0.0.0-20210728190223-81136d95d4bb
	github.com/magefile/mage v1.10.0
	github.com/pkg/profile v1.6.0
	golang.org/x/crypto v0.0.0-20210415154028-4f45737414dc
	golang.org/x/exp v0.0.0-20210722180016-6781d3edade3
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
)

replace golang.org/x/crypto => github.com/ProtonMail/crypto v0.0.0-20200605105621-11f6ee2dd602

replace gioui.org => ../../gioui/gio
replace git.sr.ht/~gioverse/skel => ../../skel
