// Code generated by 'make site/src/api/typesGenerated.ts'. DO NOT EDIT.


// PartialRecord is useful when a union string literal is a record key type but
// we do not want to fill all keys on every record.
//
// eslint-disable-next-line @typescript-eslint/no-explicit-any -- necessary for abstract record support
type PartialRecord<K extends keyof any, T> = {
	[P in K]?: T;
};

// From codersdk/enums.go
export type Enum = "bar" | "baz" | "foo" | "qux"
export const Enums: Enum[] = ["bar", "baz", "foo", "qux"]
