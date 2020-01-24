import * as util from "util"

function testClosure0() {
    let v = 0

    let f = () => { v++ }

    f()

    util.assertEqual(1, v)
}

function testClosure1() {
    let counter = () => {
        let i = 0
        return function () {
            i++
            return i
        }
    }

    let next = counter()
    let v = next()
    v += next()

    util.assertEqual(3, v)
}

function testClosure2() {
    let foo = () => {
        let a = 5
        return () => {
            return () => {
                return () => {
                    return a
                }
            }
        }
    }
    let v = foo()()()()

    util.assertEqual(5, v)
}

function testClosure3() {
    let foo = () => {
        let a = 8;
        let b = 5;
        return function () {
            let c = 2;
            return function () {
                let j = 6;
                return function () {
                    return a + b + j + c;
                }
            }
        }
    }

    let v = foo()()()()

    util.assertEqual(21, v)
}

function testClosure4() {
    let counter = () => {
        let i = 0
        return () => {
            i++
            return i
        }
    }

    let counterWrap = () => {
        let f = counter()
        return f
    }

    let next = counterWrap()
    let v = next()
    v += next()

    util.assertEqual(3, v)
}

// function testClosureClosure5() {
//     let v = 1

//     let add = () => { v++ }

//     add()

//     util.assertEqual(2, v)
// }
