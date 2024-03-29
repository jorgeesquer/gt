import * as util from "util";


function testLabelSimpleLabel() {
    let index = 0;
    label:
    for (var i = 0; i < 10; i++) {
        index++
        break label
    }
    util.assertEqual(1, index)

}

function testLabelTryNestedSameNameLabel() {
    let indice = 0;
    let indice1 = 0;
    let indice2 = 0;

    label:
    for (var i = 0; i < 3; i++) {
        indice++
        //@ts-ignore
        label:
        for (var i = 0; i < 3; i++) {
            indice1++
            break label

            indice1++
        }
        break label
        indice2++
        //break parent
    }
    util.assertEqual(1, indice)
    util.assertEqual(1, indice1)
    util.assertEqual(0, indice2)
}

function testLabelLabelAndWhilewithSwitchVariable() {
    foo:
    while (true) {
        switch ("") { case "": break foo }
    }
}

function testLabelLabelFromtryCatch() {
    let e = 0

    try {
        throw ""
    } catch (r) {
        LABEL:
        for (var i = 0; i < 5; i++) {
            try {
                throw ""
            } catch{
                break LABEL
            }
            finally {
                e++
            }
        }
    }
    finally {
        e++
    }
    util.assertEqual(2, e)
}

function testLabelLabelFromNestedtryCatch() {
    let e = 0

    try {

        try {
            throw ""
        } catch (r) {
            LABEL2:
            for (var i = 0; i < 5; i++) {
                try {
                    throw ""
                } catch{
                    break LABEL2
                }
                finally {
                    throw ""
                    e++
                }
            }
        }
        finally {
            e++
        }
    }
    catch{
    } finally {
        e++
    }

    util.assertEqual(2, e)
}