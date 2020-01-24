
/**
 * ------------------------------------------------------------------
 *  Helpers
 * ------------------------------------------------------------------
 */

export function assertEqual(expected: any, got: any, errorMsg?: string) {
    if (expected != got) {
        if (errorMsg) {
            throw errorMsg
        }
        throw fmt.sprintf("Expected %v, got %v", expected, got)
    }
}

export function assertNull(a: any) {
    if (a) {
        throw fmt.sprintf("Expected null: %v", a)
    }
}

export function assertNotNull(a: any) {
    if (!a) {
        throw fmt.sprintf("Expected not null: %v", a)
    }
}

export function assertException(msg: string, func: Function, ) {
    let success: boolean;
    try {
        func();
        success = true
        throw ""
    } catch (e) {
        if (success) {
            throw "Expected to fail"
        }

        // OK, it was expected
        if (msg && !e.toString().contains(msg)) {
            throw fmt.sprintf("Invalid exception, does not contain '%s': %s", msg, e.toString())
        }
    }
}
