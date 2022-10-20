// Code generated by 'make coder/scripts/apitypings/main.go'. DO NOT EDIT.

// From codersdk/generics.go
export interface DynamicGeneric<C extends comparable, A extends any, S extends Single> {
  readonly dynamic: GenericFields<C, A, string, S>
  readonly comparable: C
}

// From codersdk/generics.go
export interface GenericFields<C extends comparable, A extends any, T extends Custom, S extends Single> {
  readonly comparable: C
  readonly any: A
  readonly custom: T
  readonly again: T
  readonly single_constraint: S
}

// From codersdk/generics.go
export interface StaticGeneric {
  readonly static: GenericFields<string, number, number, string>
}

// From codersdk/generics.go
export type Custom = string | boolean | number | string[] | null

// From codersdk/generics.go
export type Single = string

export type comparable = boolean | number | string | any
