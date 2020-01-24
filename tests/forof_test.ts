import * as util from "util";


function testForOfForOfArrayEmpty() {
    //@ts-ignore
    var array = [];
    var i = 0;

    //@ts-ignore
    for (var value of array) {
        i += value
    }
    util.assertEqual(0, i)
}

function testForOfForOfArrayNull() {
    var array = null;
    var i = 0;

    //@ts-ignore
    for (var value of array) {
        i += value
    }
    util.assertEqual(0, i)
}

function testForOfForOfNull() {
    var i = 0;

    //@ts-ignore
    for (var value of null) {
        i += value
    }
    util.assertEqual(0, i)
}

function testForOfForOfUndefined() {
    var i = 0;

    //@ts-ignore
    for (var value of undefined) {
        i += value
    }
    util.assertEqual(0, i)
}

function testForOfNormalForOf() {
    var array = [0, 1, 2, 3];
    var i = 0;

    for (var value of array) {
        i += value
    }
    util.assertEqual(6, i)
}

function testForOfForOfWithArraytypes() {
    var array = [0, 'a', true, false, null, undefined, ,];
    var i = 0;

    for (var value of array) {
        util.assertEqual(value, array[i]);
        i++;
    }
    util.assertEqual(6, i)
}

function testForOfForOfWithException() {
    var i = 0;

    for (var value of [1]) {
        try {
            i++
            throw "exception"
        }
        catch{
            i++
        } finally {
            i++
        }
    }
    util.assertEqual(3, i)
}

function testForOfForOfWithLabel() {
    var i = 0;

    label:
    for (var value of [1, 2, 3]) {
        try {
            i++
            throw "exception"
        }
        catch{
            i++
            break label
        } finally {
            i++
        }
    }
    util.assertEqual(3, i)
}

function testForOfForOfUpdateArray() {
    var array = [0]
    var i = 0

    for (var item of array) {
        if (i > 3) {
            break
        }
        i++
        array.push(1)
        array.push(2)
    }
    util.assertEqual(1, i)
}

function testForOfForOfUpdateArray2() {
    var array = [0]
    var i = 0

    for (var x of array) {
        array.removeAt(0)
        i++
    }
    util.assertEqual(1, i)
}


function testForOfForOfNestedAndLabels() {
    var iterator = [1, 2, 3, 4]
    var loop = true;
    var i = 0;

    outer:
    while (loop) {
        loop = false;
        for (var x of iterator) {
            try {
                i++;
                continue outer;
            } catch (err) { }
        }
        i++;
    }
    util.assertEqual(1, i)
}

function testForOfForOfNestedAndLabels2() {
    var iterator = [1, 2, 3, 4]
    var loop = true;
    var i = 0;

    outer:
    while (loop) {
        loop = false;
        for (var x of iterator) {
            try {
                i++;
                throw "Ex"
            } catch (err) {
                break outer;
            }
        }
        i++;
    }
    util.assertEqual(1, i)
}

function testForOfForOfWithContinue() {
    var iterator = [1, 2, 3, 4]
    var i = 0

    for (var x of iterator) {
        try {
            i++
            continue
        } catch (err) { }

    }
    util.assertEqual(4, i)
}