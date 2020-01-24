import * as util from "util";

function testFor1() {
    let a = 0
    for (; false;) {
        a++
    }
    util.assertEqual(0, a)
}

function testFor2() {
    let a = 0
    for (var i = 0; i < 10; i++) {
        a++;
    }
    util.assertEqual(10, a)
}

function testFor3() {
    let a = 0
    let found = false
    for (var i = 0; (i < 10 && !found); i++) {
        a++;
        if (a > 5) {
            found = true
        }
    }
    util.assertEqual(6, a)
}

function testFor4() {
    let a = 0
    for (var i = 0; i < 10; i++) {
        a++;
        i++;
    }
    util.assertEqual(5, a)
}

function testFor5() {
    let a = 0
    for (; ;) {
        a++;
        if (a > 5) {
            break;
        }
    }
    util.assertEqual(6, a)
}

function testFor6() {
    let a = 0
    let b = 0;
    for (var i = 0; i < 10; i++) {
        if (i % 2 == 0) {
            b = b + i
            continue
        }
        a++
    }
    util.assertEqual(5, a)
    util.assertEqual(20, b)
}

//testing others...
function testFor7() {
    let a = 0
    for (var i = 0; i < 10; i += 2) {
        a++
    }
    util.assertEqual(5, a)
}

function testFor8() {
    let a = 0
    for (var i = 10; i > 0; i--) {
        a++
    }
    util.assertEqual(10, a)
}

function testFor9() {
    let a = 0
    for (var i = 10; i > 0; i -= 2) {
        a++
    }
    util.assertEqual(5, a)
}

function testFor10() {
    let a = 0
    for (var i = 0; i >> i; i++) {
        a++
        break
    }
    util.assertEqual(0, a)
}

function testFor11() {
    let a = 0

    for (var i = -576460752303423486; true; i--) {
        a++
        if (a >= 5) {
            break
        }
    }
    util.assertEqual(5, a)
}


function testFor12() {
    let a = 0

    for (var i = 576460752303423486; true; i++) {
        a++
        if (a >= 5) {
            break
        }
    }
    util.assertEqual(5, a)
}



//Some translate tests
function testFor20() {
    var accessed = false;
    for (var i = 0; false;) {
        accessed = true;
        break;
    }
    util.assertEqual(false, accessed)
}

function testFor21() {
    var accessed = false;
    for (var i = 0; "1";) {
        accessed = true;
        break;
    }
    util.assertEqual(true, accessed)
}

function testFor22() {
    var count = 0;
    for (var i = 0; null;) {
        count++;
    }
    util.assertEqual(0, count)
}

function testFor23() {
    var count = 0;
    for (var i = 0; false;) {
        count++;
    }
    util.assertEqual(0, count)
}

function testFor24() {
    var count = 0;
    for (var i = 0; -0;) {
        count++;
    }
    util.assertEqual(0, count)
}

function testFor25() {
    var count = 0;
    for (var i = 0; 2;) {
        count++;
        break;
    }
    util.assertEqual(1, count)
}

function testFor26() {
    let s = 0
    for (var index = 0; index < 10; index += 1) {
        if (index < 5) {
            continue;
        }
        s += index;
    }
    util.assertEqual(5 + 6 + 7 + 8 + 9, s)
}


//************************************************************************************** */
// Other kind of tests
//************************************************************************************** */

//Nested and complex for********************

function testFor30() {
    for (var i = 0; i < 10; i++) {
        i *= 2;
        if (i === 3) {
            throw "should not be here"
        }
    }
}

function testFor31() {
    let s = 0
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            s++
        }
    }
    util.assertEqual(2 * 2, s)
}

function testFor32() {
    let s = 0
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            if (j == 1) {
                continue
            }
            s++
        }
    }
    util.assertEqual(2, s)
}

function testFor33() {
    let s = 0
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            s++
            break
        }
    }
    util.assertEqual(2, s)
}

function testFor34() {
    let s = 0
    let r = 0
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            switch (j) {
                case 1: {
                    s += 4
                    break
                }
                    break
                case 2: {
                    throw "should not be here"
                }
                    break
                default:
                    break
            }
            r++
        }
    }
    util.assertEqual(8, s)
    util.assertEqual(4, r)
}

//nested with label
function testFor35() {
    let s = 0
    outer:
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            s++
            continue outer
        }
    }
    util.assertEqual(2, s)
}

function testFor36() {
    let s = 0
    outer:
    for (var i = 0; i < 2; i++) {
        for (var j = 0; j < 2; j++) {
            s++
            break outer
        }
    }
    util.assertEqual(1, s)
}

function testFor37() {
    let s = ""
    outer:
    for (var index = 0; index < 4; index += 1) {
        nested:
        for (var index_n = 0; index_n <= index; index_n++) {
            if (index * index_n == 6) {
                continue outer;
            }
            s += "" + index + index_n;
        }
    }
    util.assertEqual("0010112021223031", s)
}

function testFor38() {
    let s = ""
    outer: for (var index = 0; index < 4; index += 1) {
        nested: for (var index_n = 0; index_n <= index; index_n++) {
            if (index * index_n == 6) {
                continue;
            }
            s += "" + index + index_n;
        }
    }
    util.assertEqual("001011202122303133", s)
}

function testFor39() {
    let s = ""
    outer: for (var index = 0; index < 4; index += 1) {
        nested: for (var index_n = 0; index_n <= index; index_n++) {
            if (index * index_n == 6) {
                continue nested;
            }
            s += "" + index + index_n;
        }
    }
    util.assertEqual("001011202122303133", s)
}

//Mega nested for
function testFor40() {
    let s = ""
    for (var index0 = 0; index0 <= 1; index0++) {
        for (var index1 = 0; index1 <= index0; index1++) {
            for (var index2 = 0; index2 <= index1; index2++) {
                for (var index3 = 0; index3 <= index2; index3++) {
                    for (var index4 = 0; index4 <= index3; index4++) {
                        for (var index5 = 0; index5 <= index4; index5++) {
                            for (var index6 = 0; index6 <= index5; index6++) {
                                for (var index7 = 0; index7 <= index6; index7++) {
                                    for (var index8 = 0; index8 <= index1; index8++) {
                                        s += "" + index0 + index1 + index2 + index3 + index4 + index5 + index6 + index7 + index8 + '-';
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    util.assertEqual("000000000-100000000-110000000-110000001-111000000-111000001-111100000-111100001-111110000-111110001-111111000-111111001-111111100-111111101-111111110-111111111-", s)
}

//for with other structures

function testFor41() {
    let s = 0
    let e = 0
    try {
        for (var i = 0; i < 2; i++) {
            for (var j = 0; j < 2; j++) {
                s++
                throw "Exception"
            }
        }
    } catch{
        e++
    }

    util.assertEqual(1, s)
    util.assertEqual(1, e)
}

function testFor42() {
    let s = 0
    let e = 0
    try {
        for (var i = 0; i < 2; i++) {
            try {
                for (var j = 0; j < 2; j++) {
                    s++
                    throw "Exception"
                }
            } catch{
                e++
            }
        }
    } catch{
        throw "should not be here"
    }

    util.assertEqual(2, s)
    util.assertEqual(2, e)
}

function testFor43() {
    let s = 0
    let e = 0
    let e1 = 0
    try {
        for (var i = 0; i < 2; i++) {
            try {
                for (var j = 0; j < 2; j++) {
                    s++
                    throw "Exception"
                }
            } catch{
                e++
                throw "Exception"
            }
        }
    } catch{
        e1++
    }

    util.assertEqual(1, s)
    util.assertEqual(1, e)
    util.assertEqual(1, e1)
}

function testFor44() {
    let s = 0
    let e = 0
    try {
        for (var i = 0; i < 2; i++) {
            try {
                for (var j = 0; j < 2; j++) {
                    try {
                        s++
                        switch (s) {
                            case 1: break;
                        }
                        throw "Exception"
                    } catch{
                        e++
                        break
                    }
                }
            } catch{
                throw "should not be here"
            }
        }
    } catch{
        throw "should not be here"
    }

    util.assertEqual(2, s)
    util.assertEqual(2, e)
}

function testFor45() {
    let s = 0
    let e = 0
    let e1 = 0
    try {
        outer:
        for (var i = 0; i < 2; i++) {
            try {
                for (var j = 0; j < 2; j++) {
                    try {
                        s++
                        switch (i) {
                            case 1: break;
                            default:
                                throw "Exception"
                        }
                    } catch{
                        e++
                        continue outer
                    }
                    if (i == 1) {
                        throw "Exception"
                    }
                }
            } catch{
                e1++
            }
        }
    } catch{
        throw "should not be here"
    }

    util.assertEqual(2, s)
    util.assertEqual(1, e)
    util.assertEqual(1, e1)
}


function testFor46() {
    let s = 0
    outer:
    for (var i = 0; i < 2; i++) {
        s++
        if (s == 1) {
            break outer
            throw "should not be here"
        }
    }
}
