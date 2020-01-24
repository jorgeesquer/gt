package lib

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gtlang/gt/core"

	"github.com/gtlang/gt/lib/x/dbx"
	"github.com/gtlang/gt/lib/x/goql"
)

func init() {
	core.RegisterLib(SQL, `


declare namespace sql {
    export type DriverType = "mysql" | "sqlite3"

    /**
     * If you specify a databaseName every query will be parsed and all tables will be
     * prefixed with the database name: "SELECT foo FROM bar" will automatically be converted 
     * to "SELECT databasename.foo FROM bar". 
     * 
     * The parser understands most SQL standard but very little DDL (no ALTER TABLE yet).
     */
    export function open(driver: DriverType, connString: string, databaseName?: string): DB
	export function changeDatabase(name: string): void

    export function setLogAllQueries(value: boolean): void
    //export function setAuditAllQueries(value: boolean): void

    export let hasTransaction: boolean
    export let nestedTransactions: number
    export let driver: DriverType
    export let database: string
    export let transactionNestLevel: number

    export function exec(query: string | DQLQuery, ...params: any[]): Result
    export function execRaw(query: string, ...params: any[]): Result
    export function reader(query: string | SelectQuery, ...params: any[]): Reader
    export function query(query: string | SelectQuery, ...params: any[]): any[]
    export function queryValues(query: string | SelectQuery, ...params: any[]): any[]
    export function queryRaw(query: string | SelectQuery, ...params: any[]): any[]
    export function queryFirst(query: string | SelectQuery, ...params: any[]): any
    export function queryValue(query: string | SelectQuery, ...params: any[]): any
    export function queryValueRaw(query: string | SelectQuery, ...params: any[]): any
    export function loadTable(query: string | SelectQuery, ...params: any[]): Table
    export function beginTransaction(): void
    export function commit(force?: boolean): void
    export function rollback(): void
    export function hasTable(name: string): boolean
    export function databases(): string[]
    export function tables(): string[]
    export function columns(table: string): SchemaColumn[]
    export function setWhitelistFuncs(funcs: string[]): void

    /**
     * DB is a handle to the database.
     */
    export interface DB {
        database: string
        namespace: string
        writeAnyNamespace: boolean
		openAnyDatabase: boolean
        readOnly: boolean
        driver: DriverType
        nestedTransactions: number
		hasTransaction: boolean
		
		setMaxOpenConns(v: number): void
		setMaxIdleConns(v: number): void
		setConnMaxLifetime(d: time.Duration | number): void

        onQuery: (query: string | SelectQuery, ...params: any[]) => void
        open(name: string): DB
        close(): void

        reader(query: string | SelectQuery, ...params: any[]): Reader
        query(query: string | SelectQuery, ...params: any[]): any[]
        queryRaw(query: string | SelectQuery, ...params: any[]): any[]
        queryFirst(query: string | SelectQuery, ...params: any[]): any
        queryFirstRaw(query: string | SelectQuery, ...params: any[]): any
        queryValues(query: string | SelectQuery, ...params: any[]): any[]
        queryValuesRaw(query: string | SelectQuery, ...params: any[]): any[]
        queryValue(query: string | SelectQuery, ...params: any[]): any
    	queryValueRaw(query: string | SelectQuery, ...params: any[]): any

        loadTable(query: string | SelectQuery, ...params: any[]): Table
        loadTableRaw(query: string | SelectQuery, ...params: any[]): Table

        exec(query: string | Query, ...params: any[]): Result
        execRaw(query: string, ...params: any[]): Result

        beginTransaction(): void
        commit(): void
        rollback(): void

        hasDatabase(name: string): boolean
        hasTable(name: string): boolean
        databases(): string[]
        tables(): string[]
        columns(table: string): SchemaColumn[]
    }

    export interface SchemaColumn {
        name: string
        type: string
        size: number
        decimals: number
        nullable: boolean
    }

    export interface Reader {
        next(): boolean
        read(): any
        readValues(): any[]
        close(): void
    }

    export interface Result {
        lastInsertId: number
        rowsAffected: number
    }

    export interface Table {
        columns: Column[]
        rows: Row[]
        length: number
        page?: number
        pageSize?: number
        totalCount?: number
        items?: any
    }

    export interface Row extends Array<any> {
        [index: number]: any
        [key: string]: any
        length: number
        columns: Array<Column>
    }

    export type ColumnType = "string" | "int" | "float" | "bool" | "datetime"

    export interface Column {
        name: string
        type: ColumnType
    }

    export function parse(query: string, ...params: any[]): Query
    export function parseSelect(query: string, ...params: any[]): SelectQuery

    export function newSelect(): SelectQuery


    export function where(filter: string, ...params: any[]): SelectQuery

    export function orderBy(s: string): SelectQuery

    export interface Query {
        toSQL(format?: boolean, driver?: DriverType, escapeIdents?: boolean, ignoreNamespaces?: boolean): string
    }

    export interface DQLQuery extends Query {
        hasLimit: boolean
        hasWhere: boolean
        parameters: any[]
        where(s: string, ...params: any[]): SelectQuery
        and(s: string, ...params: any[]): SelectQuery
        and(filter: SelectQuery): SelectQuery
        or(s: string, ...params: any[]): SelectQuery
        limit(rowCount: number): SelectQuery
        limit(rowCount: number, offset: number): SelectQuery
    }

    export interface SelectQuery extends Query {
        columnsLength: number
        hasLimit: boolean
        hasFrom: boolean
        hasWhere: boolean
        hasDistinct: boolean
        hasOrderBy: boolean
        hasUnion: boolean
        hasGroupBy: boolean
        hasHaving: boolean
        parameters: any[]
        addColumns(s: string): SelectQuery
        setColumns(s: string): SelectQuery
        from(s: string): SelectQuery
        fromExpr(q: SelectQuery, alias: string): SelectQuery
        limit(rowCount: number): SelectQuery
        limit(rowCount: number, offset: number): SelectQuery
        groupBy(s: string): SelectQuery
        orderBy(s: string): SelectQuery
        where(s: string, ...params: any[]): SelectQuery
        having(s: string, ...params: any[]): SelectQuery
        and(s: string, ...params: any[]): SelectQuery
        and(filter: SelectQuery): SelectQuery
        or(s: string, ...params: any[]): SelectQuery
        join(s: string, ...params: any[]): SelectQuery
        removeParamAt(index: number): void
        removeLeftJoins(): void

        /**
         * copies all the elements of the query from the Where part.
         */
        setFilter(q: SelectQuery): void

        getFilterColumns(): string[]
    }

    export interface UpdateQuery extends Query {
        hasLimit: boolean
        hasWhere: boolean
        parameters: any[]
        where(s: string, ...params: any[]): SelectQuery
        and(s: string, ...params: any[]): SelectQuery
        and(filter: SelectQuery): SelectQuery
        or(s: string, ...params: any[]): SelectQuery
        limit(rowCount: number): SelectQuery
        limit(rowCount: number, offset: number): SelectQuery
    }

    export interface DeleteQuery extends Query {
        hasLimit: boolean
        hasWhere: boolean
        parameters: any[]
        where(s: string, ...params: any[]): SelectQuery
        and(s: string, ...params: any[]): SelectQuery
        and(filter: SelectQuery): SelectQuery
        or(s: string, ...params: any[]): SelectQuery
        limit(rowCount: number): SelectQuery
        limit(rowCount: number, offset: number): SelectQuery
    }
}


`)
}

var logAllQueries bool
var queryCount int

// var auditAllQueries bool

var SQL = []core.NativeFunction{
	core.NativeFunction{
		Name:      "sql.open",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			l := len(args)
			if l < 2 || l > 3 {
				return core.NullValue, fmt.Errorf("expected 2 or 3 parameters, got %d", l)
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("argument 1 must be a string, got %s", args[0].TypeName())
			}

			if args[1].Type != core.String {
				return core.NullValue, fmt.Errorf("argument 2 must be a string, got %s", args[1].TypeName())
			}

			driver := args[0].ToString()
			connString := args[1].ToString()

			db, err := dbx.Open(driver, connString)
			if err != nil {
				return core.NullValue, err
			}

			db.SetMaxOpenConns(500)
			db.SetMaxIdleConns(250)
			db.SetConnMaxLifetime(5 * time.Minute)

			if l == 3 {
				if args[2].Type != core.String {
					return core.NullValue, fmt.Errorf("argument 3 must be a string, got %s", args[2].TypeName())
				}
				name := args[2].ToString()
				if err := validateTenant(name); err != nil {
					return core.NullValue, err
				}
				db = db.Open(name)
			}

			ldb := newDB(db, vm)
			vm.SetGlobalFinalizer(ldb)
			return core.NewObject(ldb), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.changeDatabase",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			if !vm.HasPermission("openAnyDatabase") {
				return core.NullValue, ErrUnauthorized
			}

			db := GetContext(vm).DB
			if db == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}

			name := args[0].ToString()

			db.db.Open(name)

			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "sql.setLogAllQueries",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				if !GetContext(vm).Debug {
					return core.NullValue, ErrUnauthorized
				}
			}
			if err := ValidateArgs(args, core.Bool); err != nil {
				return core.NullValue, err
			}
			logAllQueries = args[0].ToBool()
			queryCount = 0
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "sql.setWhitelistFuncs",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgs(args, core.Array); err != nil {
				return core.NullValue, err
			}

			a := args[0].ToArray()

			goql.WhitelistFuncs = make([]string, len(a))

			for i, v := range a {
				if v.Type != core.String {
					return core.NullValue, fmt.Errorf("invalid value at index %d. It's a %s", i, v.TypeName())
				}
				goql.WhitelistFuncs[i] = v.ToString()
			}

			return core.NullValue, nil
		},
	},
	//	core.NativeFunc{
	//		Name:      "sql.setAuditAllQueries",
	//		Arguments: 1,
	//		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	//			if !vm.HasPermission("trusted") {
	//				if !GetContext(vm).Debug {
	//					return core.NullValue, ErrUnauthorized
	//				}
	//			}
	//			if err := ValidateArgs(args, core.BoolType); err != nil {
	//				return core.NullValue, err
	//			}
	//			auditAllQueries = args[0].ToBool()
	//			return core.NullValue, nil
	//		},
	//	},
	core.NativeFunction{
		Name:      "sql.newSelect",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := selectQuery{&goql.SelectQuery{}}
			return core.NewObject(s), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.parse",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 {
				return core.NullValue, fmt.Errorf("expected at least one argument, got %d", l)
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
			}

			v := args[0].ToString()

			var params []interface{}
			if l > 1 {
				params = getSqlParams(args[1:])
			}

			q, err := goql.ParseQuery(v)
			if err != nil {
				return core.NullValue, err
			}
			obj, err := getQueryObject(q, params)
			if err != nil {
				return core.NullValue, err
			}
			return obj, nil
		},
	},
	core.NativeFunction{
		Name:      "sql.parseSelect",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			switch l {
			case 0:
				s := selectQuery{&goql.SelectQuery{}}
				return core.NewObject(s), nil
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
			}

			v := args[0].ToString()

			var params []interface{}
			if l > 1 {
				params = getSqlParams(args[1:])
			}

			q, err := goql.Select(v, params...)
			if err != nil {
				return core.NullValue, err
			}

			s := selectQuery{q}
			return core.NewObject(s), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.where",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			switch l {
			case 0:
				s := selectQuery{&goql.SelectQuery{}}
				return core.NewObject(s), nil
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
			}

			v := args[0].ToString()

			var params []interface{}
			if l > 1 {
				params = make([]interface{}, l-1)
				for i, v := range args[1:] {
					params[i] = v.Export(0)
				}
			}

			q, err := goql.Where(v, params...)
			if err != nil {
				return core.NullValue, err
			}

			s := selectQuery{q}
			return core.NewObject(s), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.orderBy",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			l := len(args)
			if l == 0 || l > 2 {
				return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
			}

			v := args[0].ToString()

			q, err := goql.OrderBy(v)
			if err != nil {
				return core.NullValue, err
			}

			s := selectQuery{q}
			return core.NewObject(s), nil
		},
	},
	core.NativeFunction{
		Name: "->sql.transactionNestLevel",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return core.NewInt(s.TransactionNestLevel()), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.exec",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.exec(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.execRaw",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.execRaw(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.reader",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.reader(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.query",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.query(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryRaw",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}

			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryRaw(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryFirst",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryFirst(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryValues",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryValues(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryValuesRaw",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryValuesRaw(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryValue",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryValue(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.queryValueRaw",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.queryValueRaw(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.loadTable",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.loadTable(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.hasDatabase",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.hasDatabase(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.hasTable",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.hasTable(args, vm)
		},
	},
	core.NativeFunction{
		Name: "sql.databases",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.databases(args, vm)
		},
	},
	core.NativeFunction{
		Name: "sql.tables",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.tables(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.columns",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.columns(args, vm)
		},
	},
	core.NativeFunction{
		Name: "->sql.driver",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, nil
			}
			return core.NewString(s.db.Driver), nil
		},
	},
	core.NativeFunction{
		Name: "->sql.database",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, nil
			}
			return core.NewString(s.db.Database), nil
		},
	},
	core.NativeFunction{
		Name: "->sql.hasTransaction",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, nil
			}
			return core.NewBool(s.db.HasTransaction()), nil
		},
	},
	core.NativeFunction{
		Name: "->sql.nestedTransactions",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, nil
			}
			return core.NewInt(s.db.NestedTransactions()), nil
		},
	},
	core.NativeFunction{
		Name:      "sql.beginTransaction",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.beginTransaction(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.commit",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.commit(args, vm)
		},
	},
	core.NativeFunction{
		Name:      "sql.rollback",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			s := GetContext(vm).DB
			if s == nil {
				return core.NullValue, fmt.Errorf("no DB connection")
			}
			return s.rollback(args, vm)
		},
	},
}

func validateTenant(name string) error {
	if name == "" {
		return fmt.Errorf("invalid tenant: null")
	}

	if !IsIdent(name) {
		return fmt.Errorf("invalid tenant name. It can only contain alphanumeric values")
	}

	l := len(name)
	if l < 3 {
		return fmt.Errorf("tenant name too short. Min 3 chars")
	}

	if l > 40 {
		return fmt.Errorf("tenant name too long. Max 40 chars")
	}

	switch name {
	case "mysql", "performance", "information":
		return fmt.Errorf("invalid database name")
	}

	return nil
}

type sqlResult struct {
	result sql.Result
}

func (t sqlResult) Type() string {
	return "sql.Result"
}

func (t sqlResult) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "lastInsertId":
		i, err := t.result.LastInsertId()
		if err != nil {
			panic(err)
		}
		return core.NewInt64(i), nil
	case "rowsAffected":
		i, err := t.result.RowsAffected()
		if err != nil {
			panic(err)
		}
		return core.NewInt64(i), nil
	}
	return core.UndefinedValue, nil
}

func getRawQuery(driver string, v core.Value) (string, []interface{}, error) {
	switch v.Type {
	case core.String:
		return v.ToString(), nil, nil
	case core.Object:
		q, ok := v.ToObject().(selectQuery)
		if !ok {
			return "", nil, fmt.Errorf("expected a string or sql.SelectQuery, got %s", v.TypeName())
		}

		w := goql.NewWriter(q.query, q.query.Params, "", driver)
		w.EscapeIdents = false
		v, params, err := w.Write()
		if err != nil {
			return "", nil, err
		}
		return v, params, nil
	default:
		return "", nil, fmt.Errorf("expected a string or sql.SelectQuery, got %v", v)
	}
}

func getQueryObject(q goql.Query, params []interface{}) (core.Value, error) {
	var obj interface{}

	switch t := q.(type) {
	case *goql.SelectQuery:
		t.Params = params
		obj = selectQuery{t}

	case *goql.InsertQuery:
		t.Params = params
		obj = insertQuery{t}

	case *goql.UpdateQuery:
		t.Params = params
		obj = updateQuery{t}

	case *goql.DeleteQuery:
		t.Params = params
		obj = deleteQuery{t}

	case *goql.DropTableQuery:
		obj = dropTableQuery{t}

	case *goql.AlterDropQuery:
		obj = alterDropQuery{t}

	case *goql.DropDatabaseQuery:
		obj = dropDatabaseQuery{t}

	case *goql.AddConstraintQuery:
		obj = addConstraintQuery{t}

	case *goql.AddFKQuery:
		obj = addFKQuery{t}

	case *goql.AddColumnQuery:
		obj = addColumnQuery{t}

	case *goql.RenameColumnQuery:
		obj = renameColumnQuery{t}

	case *goql.ModifyColumnQuery:
		obj = modifyColumnQuery{t}

	case *goql.CreateDatabaseQuery:
		obj = createDatabaseQuery{t}

	case *goql.CreateTableQuery:
		obj = createTableQuery{t}

	case *goql.ShowQuery:
		obj = showQuery{t}

	default:
		return core.NullValue, fmt.Errorf("invalid query: %T", q)
	}

	return core.NewObject(obj), nil
}

func getQuery(v core.Value) (goql.Query, error) {
	switch v.Type {
	case core.String:
		return goql.ParseQuery(v.ToString())
	case core.Object:
		q, ok := v.ToObject().(selectQuery)
		if !ok {
			return nil, fmt.Errorf("expected a string or sql.SelectQuery, got %s", v.TypeName())
		}
		return q.query, nil
	default:
		return nil, fmt.Errorf("expected a string or sql.SelectQuery, got %v", v)
	}
}

func getSqlParams(args []core.Value) []interface{} {
	params := make([]interface{}, len(args))
	for i, v := range args {
		switch v.Type {
		case core.Null, core.Undefined:
			// leave it nil
		default:
			value := v.Export(0)
			switch t := value.(type) {
			case time.Time:
				value = t.UTC()
			}
			params[i] = value
		}
	}
	return params
}

func newReader(r *dbx.Reader, vm *core.VM) dbReader {
	rd := dbReader{r: r}
	vm.SetGlobalFinalizer(rd)
	return rd
}

type dbReader struct {
	r *dbx.Reader
}

func (r dbReader) Type() string {
	return "sql.Reader"
}

func (r dbReader) Close() error {
	return r.r.Close()
}

func (r dbReader) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "columns":
		cols, err := r.r.Columns()
		if err != nil {
			return core.NullValue, err
		}
		return core.NewObject(columns{cols}), nil
	}
	return core.UndefinedValue, nil
}

func (r dbReader) GetMethod(name string) core.NativeMethod {
	switch name {
	case "next":
		return r.next
	case "read":
		return r.read
	case "readValues":
		return r.readValues
	case "close":
		return r.close
	}
	return nil
}

func (r dbReader) next(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}
	return core.NewBool(r.r.Next()), nil
}

func (r dbReader) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}
	err := r.r.Close()
	return core.NullValue, err
}

func (r dbReader) read(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	cols, err := r.r.Columns()
	if err != nil {
		return core.NullValue, err
	}

	values, err := r.r.Read()
	if err != nil {
		return core.NullValue, err
	}

	obj := make(map[string]core.Value, len(cols))
	for i, col := range cols {
		obj[col.Name] = convertDBValue(values[i])
	}

	return core.NewMapValues(obj), nil
}

func (r dbReader) readValues(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	values, err := r.r.Read()
	if err != nil {
		return core.NullValue, err
	}

	vs := make([]core.Value, len(values))
	for i, v := range values {
		vs[i] = convertDBValue(v)
	}

	return core.NewArrayValues(vs), nil
}

type column struct {
	col *dbx.Column
}

func (t column) Type() string {
	return "sql.Column"
}

func (t column) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(t.col.Name), nil
	case "type":
		return core.NewString(t.col.Type.String()), nil
	}
	return core.UndefinedValue, nil
}

type columns struct {
	columns []*dbx.Column
}

func (t columns) Type() string {
	return "sql.Columns"
}

func (t columns) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "length":
		return core.NewInt(len(t.columns)), nil
	}
	return core.UndefinedValue, nil
}

func (t columns) GetIndex(i int) (core.Value, error) {
	cols := t.columns
	if i >= len(cols) {
		return core.NullValue, fmt.Errorf("index out of range")
	}

	return core.NewObject(column{cols[i]}), nil
}

func (r columns) Values() ([]core.Value, error) {
	vs := r.columns
	values := make([]core.Value, len(vs))
	for i, v := range vs {
		values[i] = core.NewObject(column{v})
	}
	return values, nil
}

type table struct {
	dbxTable   *dbx.Table
	Page       int
	PageSize   int
	TotalCount int
	Items      core.Value
}

func (t *table) Type() string {
	return "sql.Table"
}

func (t *table) Export(recursionLevel int) interface{} {
	if t.PageSize > 0 {
		return dbx.PagedTable{
			Table:      t.dbxTable,
			PageSize:   t.PageSize,
			Page:       t.Page,
			TotalCount: t.TotalCount,
			Items:      t.Items,
		}
	}

	return t.dbxTable
}

func (t *table) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "length":
		return core.NewInt(len(t.dbxTable.Rows)), nil
	case "rows":
		return core.NewObject(&rows{t.dbxTable}), nil
	case "columns":
		return core.NewObject(columns{t.dbxTable.Columns}), nil
	case "items":
		return t.Items, nil
	}
	return core.UndefinedValue, nil
}

func (t *table) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "totalCount":
		if v.Type != core.Int {
			return fmt.Errorf("invalid type. Expected an int, got %s", v.TypeName())
		}
		t.TotalCount = int(v.ToInt())
		return nil

	case "page":
		if v.Type != core.Int {
			return fmt.Errorf("invalid type. Expected an int, got %s", v.TypeName())
		}
		t.Page = int(v.ToInt())
		return nil

	case "pageSize":
		if v.Type != core.Int {
			return fmt.Errorf("invalid type. Expected an int, got %s", v.TypeName())
		}
		t.PageSize = int(v.ToInt())
		return nil

	case "items":
		t.Items = v
		return nil

	}

	return ErrReadOnlyOrUndefined
}

type rows struct {
	table *dbx.Table
}

func (r *rows) Type() string {
	return "sql.Rows"
}

func (r *rows) Len() int {
	return len(r.table.Rows)
}

func (r *rows) Export(recursionLevel int) interface{} {
	return r.table.Rows
}

func (r *rows) Values() ([]core.Value, error) {
	t := r.table
	rows := t.Rows
	values := make([]core.Value, len(rows))
	for i, v := range rows {
		values[i] = core.NewObject(newRow(v, t))
	}
	return values, nil
}

func (r *rows) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "length":
		return core.NewInt(len(r.table.Rows)), nil
	}
	return core.UndefinedValue, nil
}

func (r *rows) GetIndex(i int) (core.Value, error) {
	t := r.table

	if i >= len(t.Rows) {
		return core.NullValue, fmt.Errorf("index out of range")
	}

	return core.NewObject(newRow(t.Rows[i], t)), nil
}

func newRow(r *dbx.Row, table *dbx.Table) *row {
	return &row{table: table, dbxRow: r, mutex: &sync.RWMutex{}}
}

type row struct {
	table  *dbx.Table
	dbxRow *dbx.Row
	mutex  *sync.RWMutex
}

func (r *row) Type() string {
	return "sql.Row"
}

func (r *row) Export(recursionLevel int) interface{} {
	return r.dbxRow
}

func (r *row) Values() ([]core.Value, error) {
	vs := r.dbxRow.Values
	values := make([]core.Value, len(vs))
	for i, v := range vs {
		values[i] = convertDBValue(v)
	}
	return values, nil
}

func convertDBValue(v interface{}) core.Value {
	switch t := v.(type) {
	case time.Time:
		return core.NewObject(TimeObj(t))
	default:
		return core.NewValue(v)
	}
}

func (r *row) GetProperty(name string, vm *core.VM) (core.Value, error) {
	// first look for values from the database
	if v, ok := r.dbxRow.Value(name); ok {
		return convertDBValue(v), nil
	}

	// custom properties
	switch name {
	case "length":
		return core.NewInt(len(r.dbxRow.Values)), nil
	case "columns":
		return core.NewObject(columns{r.table.Columns}), nil
	}

	return core.UndefinedValue, nil
}

func (r *row) SetProperty(name string, v core.Value, vm *core.VM) error {
	i := r.dbxRow.ColumnIndex(name)
	if i != -1 {
		r.dbxRow.Values[i] = v.Export(0)
		return nil
	}
	return fmt.Errorf("column %s not exists", name)
}

func (r *row) SetIndex(i int, v core.Value) error {
	if i >= len(r.dbxRow.Values) {
		return fmt.Errorf("index out of range")
	}
	r.dbxRow.Values[i] = v.Export(0)
	return nil
}

func (r *row) GetIndex(i int) (core.Value, error) {
	if i >= len(r.dbxRow.Values) {
		return core.NullValue, fmt.Errorf("index out of range")
	}

	return convertDBValue(r.dbxRow.Values[i]), nil
}

func newDB(db *dbx.DB, vm *core.VM) *libDB {
	return &libDB{db: db}
}

type libDB struct {
	onQuery core.Value
	db      *dbx.DB
}

func (s *libDB) Close() error {
	if s.db.HasTransaction() {
		if err := s.db.Rollback(); err != nil {
			return err
		}
	}
	return s.db.Close()
}

func (s *libDB) Type() string {
	return "sql.DB"
}

func (s *libDB) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "hasTransaction":
		return core.NewBool(s.db.HasTransaction()), nil
	case "database":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewString(s.db.Database), nil
	case "namespace":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewString(s.db.Namespace), nil
	case "writeAnyNamespace":
		if !vm.HasPermission("writeAnyDatabaseNamespace") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewBool(s.db.WriteAnyNamespace), nil
	case "openAnyDatabase":
		if !vm.HasPermission("openAnyDatabase") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewBool(s.db.OpenAnyDatabase), nil
	case "readOnly":
		return core.NewBool(s.db.ReadOnly), nil
	case "driver":
		return core.NewString(s.db.Driver), nil
	case "onQuery":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		return s.onQuery, nil
	case "nestedTransactions":
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}
		return core.NewInt(s.db.NestedTransactions()), nil
	}
	return core.UndefinedValue, nil
}

func (s *libDB) SetProperty(name string, v core.Value, vm *core.VM) error {
	if !vm.HasPermission("trusted") {
		return ErrUnauthorized
	}

	switch name {
	case "onQuery":
		switch v.Type {
		case core.Func:
		case core.Object:
			if _, ok := v.ToObject().(core.Closure); !ok {
				return fmt.Errorf("expected a function, got: %s", v.TypeName())
			}
		default:
			return fmt.Errorf("expected a function, got: %s", v.TypeName())
		}

		s.onQuery = v
		return nil

	case "database":
		if v.Type != core.String {
			return fmt.Errorf("expected string, got %s", v.TypeName())
		}
		tenant := v.ToString()
		if err := validateTenant(tenant); err != nil {
			return err
		}
		s.db.Database = tenant
		return nil

	case "namespace":
		if v.Type != core.String {
			return fmt.Errorf("expected string, got %s", v.TypeName())
		}
		namespace := strings.Replace(v.ToString(), ".", ":", -1)
		if err := goql.ValidateNamespace(namespace); err != nil {
			return err
		}
		s.db.Namespace = namespace
		return nil

	case "writeAnyNamespace":
		if v.Type != core.Bool {
			return fmt.Errorf("expected bool, got %s", v.TypeName())
		}
		s.db.WriteAnyNamespace = v.ToBool()
		return nil

	case "openAnyDatabase":
		if v.Type != core.Bool {
			return fmt.Errorf("expected bool, got %s", v.TypeName())
		}
		s.db.OpenAnyDatabase = v.ToBool()
		return nil

	case "readOnly":
		readOnly := v.ToBool()

		if v.Type != core.Bool {
			return fmt.Errorf("expected bool, got %s", v.TypeName())
		}

		s.db.ReadOnly = readOnly
		return nil

	default:
		return ErrReadOnlyOrUndefined
	}
}

func (s *libDB) GetMethod(name string) core.NativeMethod {
	switch name {
	case "open":
		return s.open
	case "close":
		return s.close
	case "exec":
		return s.exec
	case "reader":
		return s.reader
	case "query":
		return s.query
	case "queryFirst":
		return s.queryFirst
	case "queryValue":
		return s.queryValue
	case "queryValueRaw":
		return s.queryValueRaw
	case "queryValues":
		return s.queryValues
	case "queryValuesRaw":
		return s.queryValuesRaw
	case "loadTable":
		return s.loadTable
	case "execRaw":
		return s.execRaw
	case "queryRaw":
		return s.queryRaw
	case "queryFirstRaw":
		return s.queryFirstRaw
	case "loadTableRaw":
		return s.loadTableRaw
	case "beginTransaction":
		return s.beginTransaction
	case "commit":
		return s.commit
	case "rollback":
		return s.rollback
	case "tables":
		return s.tables
	case "databases":
		return s.databases
	case "hasDatabase":
		return s.hasDatabase
	case "hasTable":
		return s.hasTable
	case "columns":
		return s.columns
	case "toSQL":
		return s.toSQL
	case "setMaxOpenConns":
		return s.setMaxOpenConns
	case "setMaxIdleConns":
		return s.setMaxIdleConns
	case "setConnMaxLifetime":
		return s.setConnMaxLifetime
	}
	return nil
}

func (s *libDB) setMaxOpenConns(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	v := args[0].ToInt()
	s.db.SetMaxOpenConns(int(v))

	return core.NullValue, nil
}

func (s *libDB) setMaxIdleConns(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	v := args[0].ToInt()
	s.db.SetMaxIdleConns(int(v))

	return core.NullValue, nil
}

func (s *libDB) setConnMaxLifetime(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgRange(args, 1, 1); err != nil {
		return core.NullValue, err
	}

	d, err := ToDuration(args[0])
	if err != nil {
		return core.NullValue, err
	}

	s.db.SetConnMaxLifetime(d)
	return core.NullValue, nil
}

func (s *libDB) open(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") && !s.db.OpenAnyDatabase {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	if err := validateTenant(name); err != nil {
		return core.NullValue, err
	}

	db := s.db.Open(name)
	ldb := newDB(db, vm)
	ldb.onQuery = s.onQuery
	return core.NewObject(ldb), nil
}

func (s *libDB) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 arguments, got %d", len(args))
	}

	if err := s.db.Close(); err != nil {
		return core.NullValue, err
	}
	return core.NullValue, nil
}

func (s *libDB) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var q goql.Query

	a := args[0]

	switch a.Type {
	case core.Object:
		var ok bool
		q, ok = a.ToObject().(goql.Query)
		if !ok {
			return core.NullValue, fmt.Errorf("expected a query, got %s", a.TypeName())
		}
	case core.String:
		var err error
		q, err = goql.ParseQuery(a.ToString())
		if err != nil {
			return core.NullValue, err
		}
	}

	sq, _, err := s.toSQLString(q, nil, vm)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewString(sq), nil
}

func (s *libDB) toSQLString(q goql.Query, params []interface{}, vm *core.VM) (string, []interface{}, error) {
	w := goql.NewWriter(q, params, s.db.Database, s.db.Driver)
	w.Location = GetContext(vm).GetLocation()
	w.Namespace = s.db.Namespace
	w.WriteAnyNamespace = s.db.WriteAnyNamespace
	w.EscapeIdents = true
	return w.Write()
}

func (s *libDB) TransactionNestLevel() int {
	return s.db.TransactionNestLevel()
}

func (s *libDB) beginTransaction(args []core.Value, vm *core.VM) (core.Value, error) {
	err := s.db.Begin()
	return core.NullValue, err
}

func (s *libDB) commit(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Bool); err != nil {
		return core.NullValue, err
	}

	var err error

	if len(args) == 1 && args[0].ToBool() {
		err = s.db.CommitForce()
	} else {
		err = s.db.Commit()
	}

	return core.NullValue, err
}

func (s *libDB) rollback(args []core.Value, vm *core.VM) (core.Value, error) {
	err := s.db.Rollback()
	return core.NullValue, err
}

func (s *libDB) execRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	var query string
	var params []interface{}
	var err error

	l := len(args)

	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	a := args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("invalid query, got %v", a)
	}
	query = a.ToString()

	if l > 1 {
		params = getSqlParams(args[1:])
	}

	res, err := s.db.ExecRaw(query, params...)
	if err != nil {
		if errors.Is(err, dbx.ErrReadOnly) {
			return core.NullValue, core.NewPublicError(err.Error())
		}
		return core.NullValue, err
	}

	return core.NewObject(sqlResult{res}), nil
}

func getExecQuery(args []core.Value, vm *core.VM) (goql.Query, []interface{}, error) {
	l := len(args)
	if l == 0 {
		return nil, nil, errors.New("no query provided")
	}

	var q goql.Query

	a := args[0]
	switch a.Type {
	case core.String:
		var err error
		q, err = goql.ParseQuery(a.ToString())
		if err != nil {
			return nil, nil, err
		}
	case core.Object:
		switch t := a.ToObject().(type) {
		case insertQuery:
			q = t.query
		case updateQuery:
			q = t.query
		case deleteQuery:
			q = t.query
		default:
			return nil, nil, errors.New("invalid query to pass parameters")
		}
	default:
		return nil, nil, fmt.Errorf("expected a query, got %s", a.TypeName())
	}

	// check permissions
	switch q.(type) {
	case *goql.InsertQuery:
	case *goql.UpdateQuery:
	case *goql.DeleteQuery:
	case *goql.CreateTableQuery:
	case *goql.AddConstraintQuery:
	case *goql.AddFKQuery:
	case *goql.AddColumnQuery:
	case *goql.RenameColumnQuery:
	case *goql.ModifyColumnQuery:
	case *goql.DropTableQuery:
	case *goql.AlterDropQuery:
		break
	default:
		if !vm.HasPermission("trusted") {
			return nil, nil, ErrUnauthorized
		}
	}

	var params []interface{}

	if l > 1 {
		params = getSqlParams(args[1:])
		switch t := q.(type) {
		case *goql.SelectQuery:
			t.Params = append(t.Params, params...)

		case *goql.InsertQuery:
			t.Params = append(t.Params, params...)

		case *goql.UpdateQuery:
			t.Params = append(t.Params, params...)

		case *goql.DeleteQuery:
			t.Params = append(t.Params, params...)

		default:
			return nil, nil, errors.New("invalid query to pass parameters")
		}
	}

	return q, params, nil
}

func (s *libDB) exec(args []core.Value, vm *core.VM) (core.Value, error) {
	q, params, err := getExecQuery(args, vm)
	if err != nil {
		return core.NullValue, err
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return core.NullValue, err
	}

	sQuery, sParams, err := s.toSQLString(q, params, vm)
	if err != nil {
		return core.NullValue, err
	}

	res, err := s.db.ExecRaw(sQuery, sParams...)
	if err != nil {
		if errors.Is(err, dbx.ErrReadOnly) {
			return core.NullValue, core.NewPublicError(err.Error())
		}
		return core.NullValue, err
	}

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(sqlResult{res}), nil
}

func (s *libDB) queryFirst(args []core.Value, vm *core.VM) (core.Value, error) {
	var q goql.Query
	var params []interface{}
	var err error

	switch len(args) {
	case 0:
		return core.NullValue, errors.New("no query provided")
	case 1:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
	default:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
		params = getSqlParams(args[1:])
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return core.NullValue, err
	}

	var rows *sql.Rows

	switch t := q.(type) {
	case *goql.SelectQuery:
		t.Params = append(t.Params, params...)
		sQuery, sParams, err := s.toSQLString(q, t.Params, vm)
		if err != nil {
			return core.NullValue, err
		}
		rows, err = s.db.QueryRaw(sQuery, sParams...)
		if err != nil {
			return core.NullValue, err
		}
	default:
		return core.NullValue, fmt.Errorf("not a select query")
	}

	defer rows.Close()

	t, _, err := dbx.ToTableLimit(rows, 1)
	if err != nil {
		return core.NullValue, err
	}

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	switch len(t.Rows) {
	case 0:
		return core.NullValue, nil
	case 1:
		r := t.Rows[0]
		m := make(map[string]core.Value, len(t.Columns))
		for k, col := range t.Columns {
			m[col.Name] = convertDBValue(r.Values[k])
		}
		return core.NewMapValues(m), nil
	default:
		panic(fmt.Sprintf("The table has more than 1 row: %d", len(t.Rows)))
	}
}

func (s *libDB) queryFirstRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	l := len(args)
	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	query, params, err := getRawQuery(s.db.Driver, args[0])
	if err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		params = append(params, getSqlParams(args[1:])...)
	}

	rows, err := s.db.QueryRaw(query, params...)
	if err != nil {
		return core.NullValue, err
	}
	if rows == nil {
		return core.NullValue, nil
	}

	defer rows.Close()

	t, err := dbx.ToTable(rows)
	if err != nil {
		return core.NullValue, err
	}

	var r *dbx.Row
	switch len(t.Rows) {
	case 0:
		return core.NullValue, nil
	case 1:
		r = t.Rows[0]
	default:
		return core.NullValue, fmt.Errorf("the query returned %d results", len(t.Rows))
	}

	m := make(map[string]core.Value, len(t.Columns))
	for k, col := range t.Columns {
		m[col.Name] = convertDBValue(r.Values[k])
	}

	return core.NewMapValues(m), nil
}

func (s *libDB) queryValue(args []core.Value, vm *core.VM) (core.Value, error) {
	var q goql.Query
	var params []interface{}
	var err error

	switch len(args) {
	case 0:
		return core.NullValue, errors.New("no query provided")
	case 1:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
	default:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
		params = getSqlParams(args[1:])
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return core.NullValue, err
	}

	var v interface{}

	switch t := q.(type) {
	case *goql.SelectQuery:
		t.Params = append(t.Params, params...)
		sQuery, sParams, err := s.toSQLString(q, t.Params, vm)
		if err != nil {
			return core.NullValue, err
		}
		v, err = s.db.QueryValueRaw(sQuery, sParams...)
		if err != nil {
			return core.NullValue, err
		}
	default:
		return core.NullValue, fmt.Errorf("not a select query")
	}

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	if v == nil {
		return core.NullValue, nil
	}
	return convertDBValue(v), nil
}

func (s *libDB) queryValueRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	l := len(args)
	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	query, params, err := getRawQuery(s.db.Driver, args[0])
	if err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		params = append(params, getSqlParams(args[1:])...)
	}

	row := s.db.QueryRowRaw(query, params...)

	var v interface{}
	if err := row.Scan(&v); err != nil {
		return core.NullValue, err
	}

	a := convertDBValue(v)
	return a, nil
}

func (s *libDB) onExecuting(q goql.Query, args []core.Value, vm *core.VM) error {
	if logAllQueries {
		v, err := toSQL(q, getSqlParams(args), []core.Value{core.TrueValue}, s.db.Namespace)
		if err != nil {
			fmt.Println("Error logging query", err)
		} else {
			fmt.Println()
			fmt.Println(v.ToString())
			fmt.Println()
		}
	}

	//	if auditAllQueries {
	//		sArgs := toValueArgs(q, args)
	//		if err := s.executeAuditAll(sArgs, vm); err != nil {
	//			return err
	//		}
	//	}

	v := s.onQuery
	switch v.Type {
	case core.Null, core.Undefined:
		return nil

	case core.Func, core.Object:
		sArgs := toValueArgs(q, args)
		return s.onQueryRawEvent(sArgs, vm)

	default:
		return fmt.Errorf("expected a function, got %v", v.TypeName())
	}
}

func (s *libDB) onExecuted(q goql.Query, args []core.Value, d time.Duration, vm *core.VM) error {
	if logAllQueries {
		queryCount++
		fmt.Printf("Duration %4f. Total %d\n", d.Seconds(), queryCount)
	}
	return nil
}

func (s *libDB) onExecutedRaw(q string, args []core.Value, d time.Duration, vm *core.VM) error {
	if logAllQueries {
		queryCount++
		fmt.Printf("Duration %4f. Total %d\n", d.Seconds(), queryCount)
	}
	return nil
}

func toValueArgs(q goql.Query, args []core.Value) []core.Value {
	var sq core.Value
	switch t := q.(type) {
	case *goql.SelectQuery:
		sq = core.NewObject(selectQuery{t})
	case *goql.InsertQuery:
		sq = core.NewObject(insertQuery{t})
	case *goql.UpdateQuery:
		sq = core.NewObject(updateQuery{t})
	case *goql.DeleteQuery:
		sq = core.NewObject(deleteQuery{t})
	default:
		sq = core.NewObject(query{t})
	}

	sArgs := []core.Value{sq}
	if len(args) > 0 {
		sArgs = append(sArgs, core.NewArrayValues(args))
	}

	return sArgs
}

func (s *libDB) onExecutingRaw(args []core.Value, vm *core.VM) error {
	if logAllQueries {
		switch args[0].Type {
		case core.String:
			fmt.Println(args[0].ToString())
		}
	}

	return s.onQueryRawEvent(args, vm)
}

func (s *libDB) onQueryRawEvent(args []core.Value, vm *core.VM) error {
	v := s.onQuery

	switch v.Type {
	case core.Null, core.Undefined:
		return nil

	case core.Func:
		if _, err := vm.RunFuncIndex(s.onQuery.ToFunction(), args...); err != nil {
			return fmt.Errorf("error in onQueryCall: %v", err)
		}
		return nil

	case core.Object:
		c, ok := v.ToObject().(core.Closure)
		if !ok {
			return fmt.Errorf("expected a function, got: %s", v.TypeName())
		}
		if _, err := vm.RunClosure(c, args...); err != nil {
			return fmt.Errorf("error in onQueryCall: %v", err)
		}
		return nil

	default:
		return fmt.Errorf("expected a function, got %v", v.TypeName())
	}
}

func (s *libDB) hasDatabase(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	exists, err := s.db.HasDatabase(args[0].ToString())
	if err != nil {
		return core.NullValue, err
	}

	return core.NewBool(exists), nil
}

func (s *libDB) hasTable(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	exists, err := s.db.HasTable(args[0].ToString())
	if err != nil {
		return core.NullValue, err
	}

	return core.NewBool(exists), nil
}

func (s *libDB) tables(args []core.Value, vm *core.VM) (core.Value, error) {
	tables, err := s.db.Tables()
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(tables))

	for i, t := range tables {
		result[i] = core.NewString(t)
	}

	return core.NewArrayValues(result), nil
}

func (s *libDB) databases(args []core.Value, vm *core.VM) (core.Value, error) {
	table, err := s.db.Databases()
	if err != nil {
		return core.NullValue, err
	}

	rows := table.Rows

	result := make([]core.Value, len(rows))

	for i, row := range rows {
		result[i] = core.NewString(row.Values[0].(string))
	}

	return core.NewArrayValues(result), nil
}

func (s *libDB) columns(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	var dbName, table string

	parts := Split(args[0].ToString(), ".")

	switch len(parts) {
	case 1:
		dbName = s.db.Database
		table = parts[0]
		if !validateTable(table) {
			return core.NullValue, fmt.Errorf("invalid table: %s", args[0])
		}

	case 2:
		if !vm.HasPermission("trusted") {
			return core.NullValue, ErrUnauthorized
		}

		dbName = parts[0]
		table = parts[1]
		if !validateTable(parts[1]) {
			return core.NullValue, fmt.Errorf("invalid table: %s", args[0])
		}

	default:
		return core.NullValue, fmt.Errorf("invalid table: %s", args[0])

	}

	if strings.ContainsRune(table, ':') {
		table = strings.Replace(table, ":", "_", -1)
	} else if s.db.Namespace != "" {
		table = strings.Replace(s.db.Namespace, ":", "_", -1) + "_" + table
	}

	if dbName != "" {
		table = dbName + "." + table
	}

	columns, err := s.db.Columns(table)
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(columns))

	for i, v := range columns {
		c, err := newColumn(v)
		if err != nil {
			return core.NullValue, err
		}

		result[i] = core.NewObject(c)
	}

	return core.NewArrayValues(result), nil
}

func validateTable(table string) bool {
	parts := Split(table, ":")

	for _, p := range parts {
		if !IsIdent(p) {
			return false
		}
	}

	return true
}

func (s *libDB) query(args []core.Value, vm *core.VM) (core.Value, error) {
	tbl, err := s.getTable(args, vm)
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(tbl.Rows))
	l := len(tbl.Columns)

	for i, r := range tbl.Rows {
		m := make(map[string]core.Value, l)
		for k, col := range tbl.Columns {
			m[col.Name] = convertDBValue(r.Values[k])
		}
		result[i] = core.NewMapValues(m)
	}

	return core.NewArrayValues(result), nil
}

type schemaColumn struct {
	name     string
	typeName string
	size     int
	decimals int
	nullable bool
}

func (schemaColumn) Type() string {
	return "sql.SchemaColumn"
}

func (c schemaColumn) String() string {
	return c.name + ":sql.SchemaColumn"
}

func (c schemaColumn) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(c.name), nil
	case "type":
		return core.NewString(c.typeName), nil
	case "nullable":
		return core.NewBool(c.nullable), nil
	case "size":
		return core.NewInt(c.size), nil
	case "decimals":
		return core.NewInt(c.decimals), nil
	}
	return core.UndefinedValue, nil
}

func newColumn(c dbx.SchemaColumn) (schemaColumn, error) {
	var size, decimals int

	t := c.Type
	i := strings.IndexRune(t, '(')

	if i != -1 {
		parts := strings.Split(t[i+1:len(t)-1], ",")
		t = t[:i]

		s, err := strconv.Atoi(parts[0])
		if err != nil {
			return schemaColumn{}, fmt.Errorf("invalid size for %s: %v", c.Name, err)
		}
		size = s

		if len(parts) > 1 {
			d, err := strconv.Atoi(parts[1])
			if err != nil {
				return schemaColumn{}, fmt.Errorf("invalid size for %s: %v", c.Name, err)
			}
			decimals = d
		}
	}

	col := schemaColumn{
		name:     c.Name,
		nullable: c.Nullable,
		typeName: t,
		size:     size,
		decimals: decimals,
	}

	return col, nil
}

func (s *libDB) queryValues(args []core.Value, vm *core.VM) (core.Value, error) {
	tbl, err := s.getTable(args, vm)
	if err != nil {
		return core.NullValue, err
	}

	cols := tbl.Columns
	if len(cols) != 1 {
		return core.NullValue, fmt.Errorf("the query must return 1 column, got %d", len(cols))
	}

	result := make([]core.Value, len(tbl.Rows))

	for i, r := range tbl.Rows {
		result[i] = convertDBValue(r.Values[0])
	}

	return core.NewArrayValues(result), nil
}

func (s *libDB) queryValuesRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	l := len(args)
	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	query, params, err := getRawQuery(s.db.Driver, args[0])
	if err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		params = append(params, getSqlParams(args[1:])...)
	}

	rows, err := s.db.QueryRaw(query, params...)
	if err != nil {
		return core.NullValue, err
	}
	if rows == nil {
		return core.NullValue, nil
	}

	defer rows.Close()

	tbl, err := dbx.ToTable(rows)
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(tbl.Rows))

	for i, r := range tbl.Rows {
		result[i] = convertDBValue(r.Values[0])
	}

	return core.NewArrayValues(result), nil
}

func (s *libDB) getTable(args []core.Value, vm *core.VM) (*dbx.Table, error) {
	var q goql.Query
	var params []interface{}
	var err error

	switch len(args) {
	case 0:
		return nil, errors.New("no query provided")
	case 1:
		if q, err = getQuery(args[0]); err != nil {
			return nil, err
		}
	default:
		if q, err = getQuery(args[0]); err != nil {
			return nil, err
		}
		params = getSqlParams(args[1:])
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return nil, err
	}

	var tbl *dbx.Table

	switch t := q.(type) {
	case *goql.ShowQuery:
		sQuery, _, err := s.toSQLString(q, nil, vm)
		if err != nil {
			return nil, err
		}
		rows, err := s.db.QueryRaw(sQuery)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		tbl, err = dbx.ToTable(rows)
		if err != nil {
			return nil, err
		}

	case *goql.SelectQuery:
		t.Params = append(t.Params, params...)
		sQuery, sParams, err := s.toSQLString(q, t.Params, vm)
		if err != nil {
			return nil, err
		}
		rows, err := s.db.QueryRaw(sQuery, sParams...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		tbl, err = dbx.ToTable(rows)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("not a select query")
	}

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return nil, err
	}

	return tbl, nil
}

func (s *libDB) queryRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	start := time.Now()

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	l := len(args)
	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	q, params, err := getRawQuery(s.db.Driver, args[0])
	if err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		params = append(params, getSqlParams(args[1:])...)
	}

	rows, err := s.db.QueryRaw(q, params...)
	if err != nil {
		return core.NullValue, err
	}
	if rows == nil {
		return core.NullValue, nil
	}

	defer rows.Close()

	t, err := dbx.ToTable(rows)
	if err != nil {
		return core.NullValue, err
	}

	result := make([]core.Value, len(t.Rows))
	l = len(t.Columns)

	for i, r := range t.Rows {
		m := make(map[string]core.Value, l)
		for k, col := range t.Columns {
			m[col.Name] = convertDBValue(r.Values[k])
		}
		result[i] = core.NewMapValues(m)
	}

	if err := s.onExecutedRaw(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	return core.NewArrayValues(result), nil
}

func (s *libDB) reader(args []core.Value, vm *core.VM) (core.Value, error) {
	var q goql.Query
	var params []interface{}
	var err error

	switch len(args) {
	case 0:
		return core.NullValue, errors.New("no query provided")
	case 1:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
	default:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
		params = getSqlParams(args[1:])
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return core.NullValue, err
	}

	var dbxReader *dbx.Reader

	switch t := q.(type) {
	case *goql.ShowQuery:
		sQuery, _, err := s.toSQLString(q, nil, vm)
		if err != nil {
			return core.NullValue, err
		}
		dbxReader, err = s.db.ReaderRaw(sQuery)
		if err != nil {
			return core.NullValue, err
		}

	case *goql.SelectQuery:
		t.Params = append(t.Params, params...)
		sQuery, sParams, err := s.toSQLString(t, t.Params, vm)
		if err != nil {
			return core.NullValue, err
		}
		dbxReader, err = s.db.ReaderRaw(sQuery, sParams...)
		if err != nil {
			return core.NullValue, err
		}

	default:
		return core.NullValue, fmt.Errorf("not a select query")
	}

	r := newReader(dbxReader, vm)

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(r), nil
}

func (s *libDB) loadTable(args []core.Value, vm *core.VM) (core.Value, error) {
	var q goql.Query
	var params []interface{}
	var err error

	switch len(args) {
	case 0:
		return core.NullValue, errors.New("no query provided")
	case 1:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
	default:
		if q, err = getQuery(args[0]); err != nil {
			return core.NullValue, err
		}
		params = getSqlParams(args[1:])
	}

	start := time.Now()

	if err := s.onExecuting(q, args[1:], vm); err != nil {
		return core.NullValue, err
	}

	var rows *sql.Rows
	switch t := q.(type) {
	case *goql.ShowQuery:
		sQuery, _, err := s.toSQLString(q, nil, vm)
		if err != nil {
			return core.NullValue, err
		}
		rows, err = s.db.QueryRaw(sQuery)
		if err != nil {
			return core.NullValue, err
		}
	case *goql.SelectQuery:
		t.Params = append(t.Params, params...)
		sQuery, sParams, err := s.toSQLString(q, t.Params, vm)
		if err != nil {
			return core.NullValue, err
		}
		rows, err = s.db.QueryRaw(sQuery, sParams...)
		if err != nil {
			return core.NullValue, err
		}
	default:
		return core.NullValue, fmt.Errorf("not a select query")
	}

	defer rows.Close()

	tbl, err := dbx.ToTable(rows)
	if err != nil {
		return core.NullValue, err
	}

	if err := s.onExecuted(q, args[1:], time.Since(start), vm); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(&table{dbxTable: tbl}), nil
}

func (s *libDB) loadTableRaw(args []core.Value, vm *core.VM) (core.Value, error) {
	if !vm.HasPermission("trusted") {
		return core.NullValue, ErrUnauthorized
	}

	if err := s.onExecutingRaw(args, vm); err != nil {
		return core.NullValue, err
	}

	l := len(args)
	if l == 0 {
		return core.NullValue, errors.New("no query provided")
	}

	query, params, err := getRawQuery(s.db.Driver, args[0])
	if err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		params = append(params, getSqlParams(args[1:])...)
	}

	rows, err := s.db.QueryRaw(query, params...)
	if err != nil {
		return core.NullValue, err
	}
	if rows == nil {
		return core.NullValue, nil
	}

	defer rows.Close()

	t, err := dbx.ToTable(rows)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(&table{dbxTable: t}), nil
}

type query struct {
	query goql.Query
}

func (s query) Type() string {
	return "sql.Query"
}

func (s query) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s query) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

func toSQL(query goql.Query, params []interface{}, args []core.Value, namespace string) (core.Value, error) {
	var driver string
	var escapeIdents bool
	var ignoreNamespaces bool
	var format bool

	l := len(args)

	if l == 0 {
		driver = "mysql"
	} else if l == 1 {
		driver = "mysql"
		if args[0].Type != core.Bool {
			return core.NullValue, fmt.Errorf("expected argument 1 to be a boolean, got %s", args[0].TypeName())
		}
		format = args[0].ToBool()
	} else {
		format = args[0].ToBool()
	}

	if l > 1 {
		if args[1].Type != core.String {
			return core.NullValue, fmt.Errorf("expected argument 2 to be a string, got %s", args[1].TypeName())
		}
		driver = args[1].ToString()
	}

	if l > 2 {
		if args[2].Type != core.Bool {
			return core.NullValue, fmt.Errorf("expected argument 3 to be a boolean, got %s", args[2].TypeName())
		}
		escapeIdents = args[2].ToBool()
	}

	if l > 3 {
		if args[3].Type != core.Bool {
			return core.NullValue, fmt.Errorf("expected argument 4 to be a boolean, got %s", args[3].TypeName())
		}
		ignoreNamespaces = args[3].ToBool()
	}

	if l > 4 {
		return core.NullValue, fmt.Errorf("expected max 4 arguments, got %d", len(args))
	}

	w := goql.NewWriter(query, params, "", driver)
	w.Namespace = namespace
	w.WriteAnyNamespace = true
	w.EscapeIdents = escapeIdents
	w.IgnoreNamespaces = ignoreNamespaces
	w.Format = format
	v, _, err := w.Write()
	if err != nil {
		return core.NullValue, err
	}

	return core.NewString(v), nil
}

type showQuery struct {
	query *goql.ShowQuery
}

func (s showQuery) Type() string {
	return "sql.ShowQuery"
}

func (s showQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s showQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type createTableQuery struct {
	query *goql.CreateTableQuery
}

func (s createTableQuery) Type() string {
	return "sql.CreateTableQuery"
}

func (s createTableQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s createTableQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type createDatabaseQuery struct {
	query *goql.CreateDatabaseQuery
}

func (s createDatabaseQuery) Type() string {
	return "sql.CreateDatabaseQuery"
}

func (s createDatabaseQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s createDatabaseQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type modifyColumnQuery struct {
	query *goql.ModifyColumnQuery
}

func (s modifyColumnQuery) Type() string {
	return "sql.ModifyColumnQuery"
}

func (s modifyColumnQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s modifyColumnQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type alterDropQuery struct {
	query *goql.AlterDropQuery
}

func (s alterDropQuery) Type() string {
	return "sql.AlterDropQuery"
}

func (s alterDropQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s alterDropQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type dropTableQuery struct {
	query *goql.DropTableQuery
}

func (s dropTableQuery) Type() string {
	return "sql.DropTableQuery"
}

func (s dropTableQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s dropTableQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type renameColumnQuery struct {
	query *goql.RenameColumnQuery
}

func (s renameColumnQuery) Type() string {
	return "sql.RenameColumnQuery"
}

func (s renameColumnQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s renameColumnQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type addColumnQuery struct {
	query *goql.AddColumnQuery
}

func (s addColumnQuery) Type() string {
	return "sql.AddColumnQuery"
}

func (s addColumnQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s addColumnQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type dropDatabaseQuery struct {
	query *goql.DropDatabaseQuery
}

func (s dropDatabaseQuery) Type() string {
	return "sql.DropDatabaseQuery"
}

func (s dropDatabaseQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s dropDatabaseQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type addConstraintQuery struct {
	query *goql.AddConstraintQuery
}

func (s addConstraintQuery) Type() string {
	return "sql.AddConstraintQuery"
}

func (s addConstraintQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s addConstraintQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type addFKQuery struct {
	query *goql.AddFKQuery
}

func (s addFKQuery) Type() string {
	return "sql.AddFKQuery"
}

func (s addFKQuery) String() string {
	v, err := toSQL(s.query, nil, nil, "")
	if err != nil {
		return err.Error()
	}
	return v.ToString()
}

func (s addFKQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s addFKQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, nil, args, "")
}

type deleteQuery struct {
	query *goql.DeleteQuery
}

func (s deleteQuery) Type() string {
	return "sql.DeleteQuery"
}

func (s deleteQuery) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "hasWhere":
		return core.NewBool(s.query.WherePart != nil), nil
	case "hasLimit":
		return core.NewBool(s.query.LimitPart != nil), nil
	case "parameters":
		return core.NewArrayValues(s.getParamers()), nil
	}
	return core.UndefinedValue, nil
}

func (s deleteQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "where":
		return s.where
	case "and":
		return s.and
	case "or":
		return s.or
	case "limit":
		return s.limit
	case "toSQL":
		return s.toSQL
	case "join":
		return s.join
	}
	return nil
}

func (s deleteQuery) getParamers() []core.Value {
	result := make([]core.Value, len(s.query.Params))

	for i, v := range s.query.Params {
		result[i] = core.NewValue(v)
	}

	return result
}

func (s deleteQuery) or(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Or(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s deleteQuery) and(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}

	filter := args[0]

	// the filter can be a query object
	if filter.Type == core.Object {
		f, ok := filter.ToObject().(selectQuery)
		if !ok {
			return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
		}

		// when passing a query object the parameters are contained in the object
		if l > 1 {
			return core.NullValue, fmt.Errorf("expected only 1 argument, got %d", l)
		}
		s.query.AndQuery(f.query)
		return core.NewObject(s), nil
	}

	// If its not an object the filter must be a string
	if filter.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
	}

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	v := filter.ToString()

	if err := s.query.And(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s deleteQuery) where(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Where(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s deleteQuery) limit(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 || l > 2 {
		return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
	}
	if args[0].Type != core.Int {
		// if the argument is null clear the limit
		if args[0].Type == core.Null {
			s.query.LimitPart = nil
			return core.NewObject(s), nil
		}
		return core.NullValue, fmt.Errorf("expected argument 1 to be a int, got %d", args[0].Type)
	}

	if l == 2 {
		if args[1].Type != core.Int {
			return core.NullValue, fmt.Errorf("expected argument 2 to be a int, got %d", args[1].Type)
		}
	}

	switch l {
	case 1:
		s.query.Limit(int(args[0].ToInt()))
	case 2:
		s.query.LimitOffset(int(args[0].ToInt()), int(args[1].ToInt()))
	}

	return core.NewObject(s), nil
}

func (s deleteQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, s.query.Params, args, "")
}

func (s deleteQuery) join(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := args[0].ToString()

	if err := s.query.Join(v); err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		if len(s.query.Params) > 0 {
			return core.NullValue, fmt.Errorf("can't add parameters if there are any other. TODO: fix this to allow it")
		}
		params := make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
		s.query.Params = append(s.query.Params, params...)
	}
	return core.NewObject(s), nil
}

type insertQuery struct {
	query *goql.InsertQuery
}

func (s insertQuery) Type() string {
	return "sql.InsertQuery"
}

func (s insertQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "toSQL":
		return s.toSQL
	}
	return nil
}

func (s insertQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, s.query.Params, args, "")
}

type updateQuery struct {
	query *goql.UpdateQuery
}

func (s updateQuery) Type() string {
	return "sql.UpdateQuery"
}

func (s updateQuery) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "hasWhere":
		return core.NewBool(s.query.WherePart != nil), nil
	case "hasLimit":
		return core.NewBool(s.query.LimitPart != nil), nil
	case "parameters":
		return core.NewArrayValues(s.getParamers()), nil
	}
	return core.UndefinedValue, nil
}

func (s updateQuery) getParamers() []core.Value {
	result := make([]core.Value, len(s.query.Params))

	for i, v := range s.query.Params {
		result[i] = core.NewValue(v)
	}

	return result
}

func (s updateQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "where":
		return s.where
	case "and":
		return s.and
	case "or":
		return s.or
	case "limit":
		return s.limit
	case "toSQL":
		return s.toSQL
	case "addColumns":
		return s.addColumns
	case "setColumns":
		return s.setColumns
	case "join":
		return s.join
	}
	return nil
}

func (s updateQuery) or(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Or(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s updateQuery) and(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}

	filter := args[0]

	// the filter can be a query object
	if filter.Type == core.Object {
		f, ok := filter.ToObject().(selectQuery)
		if !ok {
			return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
		}

		// when passing a query object the parameters are contained in the object
		if l > 1 {
			return core.NullValue, fmt.Errorf("expected only 1 argument, got %d", l)
		}
		s.query.AndQuery(f.query)
		return core.NewObject(s), nil
	}

	// If its not an object the filter must be a string
	if filter.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
	}

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	v := filter.ToString()

	if err := s.query.And(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s updateQuery) where(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Where(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s updateQuery) limit(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 || l > 2 {
		return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
	}
	if args[0].Type != core.Int {
		// if the argument is null clear the limit
		if args[0].Type == core.Null {
			s.query.LimitPart = nil
			return core.NewObject(s), nil
		}
		return core.NullValue, fmt.Errorf("expected argument 1 to be a int, got %d", args[0].Type)
	}

	if l == 2 {
		if args[1].Type != core.Int {
			return core.NullValue, fmt.Errorf("expected argument 2 to be a int, got %d", args[1].Type)
		}
	}

	switch l {
	case 1:
		s.query.Limit(int(args[0].ToInt()))
	case 2:
		s.query.LimitOffset(int(args[0].ToInt()), int(args[1].ToInt()))
	}

	return core.NewObject(s), nil
}

func (s updateQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, s.query.Params, args, "")
}

func (s updateQuery) addColumns(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := args[0].ToString()

	if err := s.query.AddColumns(v); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s updateQuery) setColumns(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]

	switch a.Type {
	case core.String:
		if err := s.query.SetColumns(a.ToString()); err != nil {
			return core.NullValue, err
		}
	case core.Null:
		s.query.Columns = nil
	default:
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", a.Type)
	}

	return core.NewObject(s), nil
}

func (s updateQuery) join(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := args[0].ToString()

	if err := s.query.Join(v); err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		if len(s.query.Params) > 0 {
			return core.NullValue, fmt.Errorf("can't add parameters if there are any other. TODO: fix this to allow it")
		}
		params := make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
		s.query.Params = append(s.query.Params, params...)
	}
	return core.NewObject(s), nil
}

type selectQuery struct {
	query *goql.SelectQuery
}

func (s selectQuery) Type() string {
	return "sql.SelectQuery"
}

func (s selectQuery) String() string {
	v, err := toSQL(s.query, nil, nil, "")
	if err != nil {
		return err.Error()
	}
	return v.ToString()
}

func (s selectQuery) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "columnsLength":
		return core.NewInt(len(s.query.Columns)), nil
	case "hasLimit":
		return core.NewBool(s.query.LimitPart != nil), nil
	case "hasFrom":
		return core.NewBool(s.query.From != nil), nil
	case "hasWhere":
		return core.NewBool(s.query.WherePart != nil), nil
	case "hasDistinct":
		return core.NewBool(s.query.Distinct), nil
	case "hasOrderBy":
		return core.NewBool(s.query.OrderByPart != nil), nil
	case "hasUnion":
		return core.NewBool(s.query.UnionPart != nil), nil
	case "hasGroupBy":
		return core.NewBool(s.query.GroupByPart != nil), nil
	case "hasHaving":
		return core.NewBool(s.query.HavingPart != nil), nil
	case "parameters":
		return core.NewArrayValues(s.getParamers()), nil
	}
	return core.UndefinedValue, nil
}

func (s selectQuery) getParamers() []core.Value {
	result := make([]core.Value, len(s.query.Params))

	for i, v := range s.query.Params {
		result[i] = core.NewValue(v)
	}

	return result
}

func (s selectQuery) GetMethod(name string) core.NativeMethod {
	switch name {
	case "addColumns":
		return s.addColumns
	case "setColumns":
		return s.setColumns
	case "from":
		return s.from
	case "fromExpr":
		return s.fromExpr
	case "limit":
		return s.limit
	case "orderBy":
		return s.orderBy
	case "where":
		return s.where
	case "having":
		return s.having
	case "and":
		return s.and
	case "or":
		return s.or
	case "join":
		return s.join
	case "groupBy":
		return s.groupBy
	case "setFilter":
		return s.setFilter
	case "toSQL":
		return s.toSQL
	case "getFilterColumns":
		return s.getFilterColumns
	case "removeParamAt":
		return s.removeParamAt
		// **DEPRECATE**
	case "removeLeftJoins":
		return s.removeLeftJoins
	}
	return nil
}

// **DEPRECATE**
func (s selectQuery) removeLeftJoins(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	s.query.RemoveLeftJoins()

	return core.NullValue, nil
}

func (s selectQuery) removeParamAt(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Int); err != nil {
		return core.NullValue, err
	}

	i := args[0].ToInt()
	ln := len(s.query.Params) - 1
	if i < 0 || int(i) > ln {
		return core.NullValue, fmt.Errorf("index out of range: %d, valid range is 0-%d", i, ln)
	}

	s.query.Params = append(s.query.Params[:i], s.query.Params[i+1:]...)
	return core.NullValue, nil
}

func (s selectQuery) getFilterColumns(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 0 {
		return core.NullValue, fmt.Errorf("expected 0 argument, got %d", len(args))
	}

	if s.query.WherePart == nil {
		return core.NewArray(0), nil
	}

	cols := goql.NameExprColumns(s.query.WherePart.Expr)

	vals := make([]core.Value, len(cols))

	for i, col := range cols {
		vals[i] = core.NewString(col.Name)
	}

	return core.NewArrayValues(vals), nil
}

// setFilter copies all the elements of the query from the Where part
func (s selectQuery) setFilter(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]
	var o selectQuery

	switch a.Type {
	case core.Null, core.Undefined:
		// setting it null clears the filter
		s.query.WherePart = nil
		return core.NewObject(s), nil

	case core.Object:
		var ok bool
		o, ok = a.ToObject().(selectQuery)
		if !ok {
			return core.NullValue, fmt.Errorf("expected argument 1 to be a sql.SelectQuery, got %v", a.Type)
		}
	default:
		return core.NullValue, fmt.Errorf("expected argument 1 to be a sql.SelectQuery, got %v", a.Type)
	}

	dst := s.query
	src := o.query

	dst.WherePart = src.WherePart

	if src.OrderByPart != nil {
		dst.OrderByPart = src.OrderByPart
	}

	if src.GroupByPart != nil {
		dst.GroupByPart = src.GroupByPart
	}

	if src.HavingPart != nil {
		dst.HavingPart = src.HavingPart
	}

	if src.LimitPart != nil {
		dst.LimitPart = src.LimitPart
	}

	if src.UnionPart != nil {
		dst.UnionPart = src.UnionPart
	}

	// replace all the previous parameters
	dst.Params = src.Params

	return core.NewObject(s), nil
}

func (s selectQuery) toSQL(args []core.Value, vm *core.VM) (core.Value, error) {
	return toSQL(s.query, s.query.Params, args, "")
}

func (s selectQuery) groupBy(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]

	if a.IsNil() {
		s.query.GroupByPart = nil
		return core.NewObject(s), nil
	}

	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := a.ToString()

	if err := s.query.GroupBy(v); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) join(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := args[0].ToString()

	if err := s.query.Join(v); err != nil {
		return core.NullValue, err
	}

	if l > 1 {
		if len(s.query.Params) > 0 {
			return core.NullValue, fmt.Errorf("can't add parameters if there are any other. TODO: fix this to allow it")
		}
		params := make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
		s.query.Params = append(s.query.Params, params...)
	}
	return core.NewObject(s), nil
}

func (s selectQuery) or(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Or(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) and(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}

	filter := args[0]

	// the filter can be a query object
	if filter.Type == core.Object {
		f, ok := filter.ToObject().(selectQuery)
		if !ok {
			return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
		}

		// when passing a query object the parameters are contained in the object
		if l > 1 {
			return core.NullValue, fmt.Errorf("expected only 1 argument, got %d", l)
		}
		s.query.AndQuery(f.query)
		return core.NewObject(s), nil
	}

	// If its not an object the filter must be a string
	if filter.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string or query, got %s", args[0].TypeName())
	}

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	v := filter.ToString()

	if err := s.query.And(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) where(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Where(v, params...); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) orderBy(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]

	if a.IsNil() {
		s.query.OrderByPart = nil
		return core.NewObject(s), nil
	}

	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := a.ToString()

	if err := s.query.OrderBy(v); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) having(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected at least 1 argument, got 0")
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := args[0].ToString()

	var params []interface{}
	if l > 1 {
		params = make([]interface{}, l-1)
		for i, v := range args[1:] {
			params[i] = v.Export(0)
		}
	}

	if err := s.query.Having(v, params...); err != nil {
		return core.NullValue, err
	}

	return core.NewObject(s), nil
}

func (s selectQuery) fromExpr(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object, core.String); err != nil {
		return core.NullValue, err
	}

	sel, ok := args[0].ToObjectOrNil().(selectQuery)
	if !ok {
		return core.NullValue, fmt.Errorf("expected select expression, got %T", args[0].ToObjectOrNil())
	}

	alias := args[1].ToString()

	parenExp := &goql.ParenExpr{X: sel.query}

	exp := &goql.FromAsExpr{From: parenExp, Alias: alias}

	s.query.From = append(s.query.From, exp)
	s.query.Params = append(s.query.Params, sel.query.Params...)

	return core.NewObject(s), nil
}

func (s selectQuery) from(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}

	v := "from " + args[0].ToString()

	if err := s.query.SetFrom(v); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) addColumns(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].Type)
	}
	v := args[0].ToString()

	if err := s.query.AddColumns(v); err != nil {
		return core.NullValue, err
	}
	return core.NewObject(s), nil
}

func (s selectQuery) setColumns(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	a := args[0]

	switch a.Type {
	case core.String:
		if err := s.query.SetColumns(a.ToString()); err != nil {
			return core.NullValue, err
		}
	case core.Null:
		s.query.Columns = nil
	default:
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", a.Type)
	}

	return core.NewObject(s), nil
}

func (s selectQuery) limit(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 || l > 2 {
		return core.NullValue, fmt.Errorf("expected 1 or 2 arguments, got %d", len(args))
	}
	if args[0].Type != core.Int {
		// if the argument is null clear the limit
		if args[0].Type == core.Null {
			s.query.LimitPart = nil
			return core.NewObject(s), nil
		}
		return core.NullValue, fmt.Errorf("expected argument 1 to be a int, got %d", args[0].Type)
	}

	if l == 2 {
		if args[1].Type != core.Int {
			return core.NullValue, fmt.Errorf("expected argument 2 to be a int, got %d", args[1].Type)
		}
	}

	switch l {
	case 1:
		s.query.Limit(int(args[0].ToInt()))
	case 2:
		s.query.LimitOffset(int(args[0].ToInt()), int(args[1].ToInt()))
	}

	return core.NewObject(s), nil
}
