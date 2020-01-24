import * as util from "util"

function testSlice1() {
    let a = []
    a.push(2)
    util.assertEqual(1, a.length)
    util.assertEqual(2, a[0])
}

function testSlice2() {
    let a = []
    a.push(2, 3)
    util.assertEqual(2, a.length)
    util.assertEqual(2, a[0])
    util.assertEqual(3, a[1])
}

function testSlice3() {
    let a = [2, 3]
    util.assertEqual(2, a.length)
    util.assertEqual(2, a[0])
    util.assertEqual(3, a[1])
}

function testSlice4() {
    let a = [2, 3]
    a.insertAt(1, 6)
    util.assertEqual(3, a.length)
    util.assertEqual(2, a[0])
    util.assertEqual(6, a[1])
    util.assertEqual(3, a[2])
}

function testSlice5() {
    let a = [2, 3]
    a.removeAt(1)
    util.assertEqual(1, a.length)
    util.assertEqual(2, a[0])
}

function testSlice6() {
    let a = [1]
    a.pushRange([2, 3])
    util.assertEqual(3, a.length)
    util.assertEqual(1, a[0])
    util.assertEqual(2, a[1])
    util.assertEqual(3, a[2])
}