
function testNumber1() {
    let tests: any = {
        test1: { exp: 1_000_000, r: 1000000 },
        test2: { exp: 1_0_0_0, r: 1000 },
    }

    for (let key in tests) {
        let t = tests[key]
        if (t.exp != t.r) {
            throw key + ": Expected " + t.r + " but got " + t.exp
        }
    }
}
