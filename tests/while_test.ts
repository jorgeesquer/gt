import * as util from "util";

function testWhile1() {
    let a = 0
    while (false) {
        a++
    }
    util.assertEqual(0, a)
}

function testWhile2() {
    let a = 0
    while (true) {
        a++;
        break;
    }
    util.assertEqual(1, a)
}

function testWhile3() {
    let a = 0
    while (true) {
        a++;
        if (a == 2) {
            break;
        }
        else {
            continue;
        }
        a++;
    }
    util.assertEqual(2, a)
}

function testWhile4() {
    let a = 0
    while (a < 3) {
        a++;
        while (true) {
            break;
        }
    }
    util.assertEqual(3, a)
}

function testWhile6() {
    var i = 0;
    outer:
    while (true) {
        i++;
        if (i == 10) {
            break outer
            throw "should not be here 1"
        }
    }
    util.assertEqual(10, i)
}

function testWhile7() {
    var i = 0;
    var e = 0;
    var f = 0;
    while (true) {
        try {
            i++
            throw "Test bucle"
        }
        catch{
            e++
        } finally {
            f++
        }
        if (i >= 3) {
            break;
        }
    }
    util.assertEqual(3, i)
    util.assertEqual(3, e)
    util.assertEqual(3, f)
}

function testWhile8() {
    var i = 0;
    var e = 0;
    var f = 0;
    while (true) {
        try {
            i++
            throw "Test bucle"
        }
        catch{
            e++
            break;
        } finally {
            f++
        }

    }
    util.assertEqual(1, i)
    util.assertEqual(1, e)
    util.assertEqual(1, f)
}

function testWhile9() {
    var i = 0;
    var e = 0;
    var f = 0;
    while (true) {
        try {
            i++
        }
        catch{
            e++
        } finally {
            f++
        }
        break;
    }
    util.assertEqual(1, i)
    util.assertEqual(0, e)
    util.assertEqual(1, f)
}

function testWhile10() {
    var i = 0
    var e = 0
    var f = 0
    while (true) {
        for (var j = 0; j < 3; j++) {
            f++
            break
        }
        i++
        break
    }
    util.assertEqual(1, i)
    util.assertEqual(1, f)
}

function testWhile11() {
    var i = 0
    var e = 0
    var f = 0
    while (true) {
        for (var j = 0; j < 3; j++) {
            f++
            continue
        }
        i++
        break
    }
    util.assertEqual(1, i)
    util.assertEqual(3, f)
}

function testWhile12() {
    var i = 0
    while (true) {
        switch (true) {
            case true:
                break;
        }
        i++
        break
    }
    util.assertEqual(1, i)
}

function testWhile13() {
    var i = 0
    var j = 0
    var e = 0
    try {
        while (true) {
            try {
                i++
                throw "test"
            } finally {
                j++
            }
        }
    } catch{
        e++
    } finally {
        j++
    }
    util.assertEqual(1, i)
    util.assertEqual(2, j)
    util.assertEqual(1, e)
}


function testWhile14() {
    var i = 0
    while (true) {
        i++
        while (true) {
            i++
            while (true) {
                i++
                break
            }
            break
        }
        break
    }

    util.assertEqual(3, i)
}