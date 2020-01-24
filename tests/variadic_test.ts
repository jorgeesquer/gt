import * as util from "util"

function testVariadicLen() {
    util.assertEqual(1, len(1))
    util.assertEqual(3, len(1, 2, 3))
}

function testVariadicLen2() {
    let a = [1, 2, 3]
    util.assertEqual(3, len(...a))
}

function testVariadicLen3() {
    let a = [1, 2, 3]
    util.assertEqual(5, len(1, 2, ...a))
}

function testVariadicSum1() {
    let a = [1, 2, 3]
    util.assertEqual(6, sum(...a))
}

function testVariadicSum2() {
    let a = [1, 2, 3]
    util.assertEqual(9, sum(1, 2, ...a))
}

function len(...a: number[]) {
    return a.length
}

function sum(...a: number[]) {
    let v = 0
    for (let i = 0, l = a.length; i < l; i++) {
        v += a[i]
    }
    return v
}