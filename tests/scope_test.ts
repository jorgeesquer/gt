import * as util from "util"

function testScope1() {
    let a = 3
    {
        let a = 7
        util.assertEqual(7, a)
        a++
        {
            let a = 99
            util.assertEqual(99, a)
            a++
            util.assertEqual(100, a)
        }
        util.assertEqual(8, a)
    }
    util.assertEqual(3, a)
}

function testScope2() {
    let a = 3
    {
        let a = 7
        a++
        util.assertEqual(8, a)
    }
    util.assertEqual(3, a)
}

function testScope3() {
    let a = 3
    {
        let a = 7
        {
            a++
        }
    }
    util.assertEqual(3, a)
}

function testScope4() {
    let a = 3
    if (a == 3) {
        let a = 7
        util.assertEqual(7, a)
        a++
        util.assertEqual(8, a)
    }
    util.assertEqual(3, a)
}
