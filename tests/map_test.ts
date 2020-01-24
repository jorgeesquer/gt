import * as util from "util"

function testMap1() {
    let a = { foo: { bar: 3 } }
    util.assertEqual(3, a.foo.bar)
}

function testMapDelete() {
    let a = { foo: 1 }
    map.deleteKey(a, "foo")
    util.assertEqual(undefined, a.foo)
}

function testMap3() {
    let a = { foo: 1 }
    let b = map.clone(a)

    b.foo++

    util.assertEqual(2, b.foo)
    util.assertEqual(1, a.foo)
}


function testMapBasicMap() {
    let a = { "0": 0, "1": 1, "2": 2, "3": 3, "4": 4 }
    let sum

    for (var i = 0; i < map.len(a); i++) {
        //@ts-ignore
        sum = sum + a[i]
    }
    util.assertEqual(10, sum)
}


function testMapGetMapValues() {
    let a = { "0": 0, "1": 1, "2": 2, "3": 3, "4": 4 }
    let val

    for (var i = 0; i < map.len(a); i++) {
        val = map.values(a)
    }
    util.assertEqual(5, val.length)
}


function testMapMapOverFlow() {
    let a = { "0": 0, "1": 1, "2": 2, "3": 3, "4": 4 }
    let e

    for (var i = 0; i <= map.len(a); i++) {
        //@ts-ignore
        e = a[i]
    }

    util.assertEqual(undefined, e)
}

