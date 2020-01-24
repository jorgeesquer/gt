import * as util from "util";

function testForIn1() {
    let sum = 0;
    var someArray = [1, 2, 3];
    for (var item in someArray) {
        sum = sum + someArray[item]
    }
    util.assertEqual(6, sum)
}

function testForIn2() {
    let i = 0
    for (var x in [1, null, 3, , 4]) {
        i++
    }
    util.assertEqual(4, i)
}

function testForIn3() {
    var obj = { a: 1, b: 2, c: 3 };
    var sum = 0

    for (var prop in obj) {
        sum++
    }
    util.assertEqual(3, sum)
}


function testForIn4() {
    var sum = 0
    for (var x in [1, 2, , , , 3]) {
        sum++
    }
    util.assertEqual(3, sum)
}
