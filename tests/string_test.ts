import * as util from "util"

function testStringUTF8() {
    let a = "会意字";

    util.assertEqual(9, a.length)
    util.assertEqual(3, a.runeCount)

    util.assertEqual(0xE4, a[0])
    util.assertEqual(0xBC, a[1])

    util.assertEqual("意", a.runeSubstring(1, 2))

    // starts at byte 3
    util.assertEqual(3, a.indexOf("意"))
}

function testString1() {
    let a = "asdf"
    let tests: any = [
        { exp: a.hasPrefix("asdf"), r: true },
        { exp: a.hasPrefix("a"), r: true },
        { exp: a.hasPrefix(""), r: true },
        { exp: a.hasPrefix("b"), r: false },
        { exp: a.hasPrefix("asdf9"), r: false },
        { exp: a.hasSuffix(""), r: true },
        { exp: a.hasSuffix("df"), r: true },
        { exp: a.hasSuffix("dft"), r: false },
        { exp: a.indexOf("s"), r: 1 },
        { exp: a.indexOf("x"), r: -1 },
        { exp: a.lastIndexOf("sd"), r: 1 },
        { exp: a.contains("sd"), r: true },
        { exp: a.contains("y"), r: false },
        { exp: a.equalFold("ASDf"), r: true },
    ]

    for (let i = 0, l = tests.length; i < l; i++) {
        let t = tests[i];
        if (t.exp != t.r) {
            throw fmt.sprintf("Test %d (base 1): Expected %v, got %v", i + 1, t.r, t.exp)
        }
    }
}

function testStringStringBytes() {
    let s = "el próximo año";
    let key = "año"
    let i = s.indexOf(key)
    util.assertEqual(4, key.length) // multibyte rune ñ
    util.assertEqual("año", s.substring(i, i + key.length))
}
