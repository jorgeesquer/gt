import * as util from "util"

function testConstructor() {
    let p = new Person("John", 33)
    util.assertEqual("John", p.name)
    util.assertEqual(33, p.age)
    util.assertEqual("John", p.getName())
    util.assertEqual(33, p.getAge())
}

class Person {
    name: string
    age: number

    constructor(name: string, age: number) {
        this.name = name
        this.age = age
    }

    getName() {
        return this.name
    }

    getAge() {
        return this.age
    }
}