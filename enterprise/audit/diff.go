package audit

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/google/uuid"

	"github.com/coder/coder/coderd/audit"
	"github.com/coder/coder/coderd/database"
)

func structName(t reflect.Type) string {
	return t.PkgPath() + "." + t.Name()
}

type FieldDiff struct {
	FieldType reflect.StructField
	LeftF     reflect.Value
	RightF    reflect.Value
}

func flattenStructFields(left, right any) []FieldDiff {
	leftV := reflect.ValueOf(left)
	rightV := reflect.ValueOf(right)

	allFields := []FieldDiff{}
	rightT := rightV.Type()

	// Flatten the structure and all fields.
	// Does not support named nested structs.
	for i := 0; i < rightT.NumField(); i++ {
		if !rightT.Field(i).IsExported() {
			continue
		}

		var (
			leftF  = leftV.Field(i)
			rightF = rightV.Field(i)
		)

		if rightT.Field(i).Anonymous {
			// Loop through anonymous type for fields,
			// append as top level fields for diffs.
			allFields = append(allFields, flattenStructFields(leftF.Interface(), rightF.Interface())...)
			continue
		}

		// Single fields append as is.
		allFields = append(allFields, FieldDiff{
			LeftF:     leftF,
			RightF:    rightF,
			FieldType: rightT.Field(i),
		})
	}
	return allFields
}

func diffValues(left, right any, table Table) audit.Map {
	var (
		baseDiff = audit.Map{}
		rightT   = reflect.TypeOf(right)

		diffKey = table[structName(rightT)]
	)

	if diffKey == nil {
		panic(fmt.Sprintf("dev error: type %q (type %T) attempted audit but not auditable", rightT.Name(), right))
	}

	allFields := flattenStructFields(left, right)
	fmt.Println("AllFields", allFields)
	for _, field := range allFields {
		var (
			leftF  = field.LeftF
			rightF = field.RightF

			leftI  = leftF.Interface()
			rightI = rightF.Interface()
		)

		// This is the field that is returning a blank string.
		fmt.Printf("Number of fields for %s: %d\n", rightT.String(), rightT.NumField())
		// rightT.Field(i)

		var (
			diffName = field.FieldType.Tag.Get("json")
		)
		// fmt.Println("rightT.Field(i)", rightT, rightT.Field(i), rightT.Field(i).Tag.Get("json"))

		// map[avatar_url:track id:track members:track name:track organization_id:ignore quota_allowance:track]
		fmt.Println("DIFF KEY", diffKey)

		// group
		fmt.Println("DIFF NAME", diffName)
		atype, ok := diffKey[diffName]
		if !ok {
			panic(fmt.Sprintf("dev error: field %q lacks audit information", diffName))
		}

		if atype == ActionIgnore {
			continue
		}

		// coerce struct types that would produce bad diffs.
		if leftI, rightI, ok = convertDiffType(leftI, rightI); ok {
			leftF, rightF = reflect.ValueOf(leftI), reflect.ValueOf(rightI)
		}

		// If the field is a pointer, dereference it. Nil pointers are coerced
		// to the zero value of their underlying type.
		if leftF.Kind() == reflect.Ptr && rightF.Kind() == reflect.Ptr {
			leftF, rightF = derefPointer(leftF), derefPointer(rightF)
			leftI, rightI = leftF.Interface(), rightF.Interface()
		}

		if !reflect.DeepEqual(leftI, rightI) {
			switch atype {
			case ActionTrack:
				baseDiff[diffName] = audit.OldNew{Old: leftI, New: rightI}
			case ActionSecret:
				baseDiff[diffName] = audit.OldNew{
					Old:    reflect.Zero(rightF.Type()).Interface(),
					New:    reflect.Zero(rightF.Type()).Interface(),
					Secret: true,
				}
			}
		}
	}

	return baseDiff
}

// convertDiffType converts external struct types to primitive types.
//
//nolint:forcetypeassert
func convertDiffType(left, right any) (newLeft, newRight any, changed bool) {
	switch typedLeft := left.(type) {
	case uuid.UUID:
		typedRight := right.(uuid.UUID)

		// Automatically coerce Nil UUIDs to empty strings.
		outLeft := typedLeft.String()
		if typedLeft == uuid.Nil {
			outLeft = ""
		}

		outRight := typedRight.String()
		if typedRight == uuid.Nil {
			outRight = ""
		}

		return outLeft, outRight, true

	case uuid.NullUUID:
		leftStr, _ := typedLeft.MarshalText()
		rightStr, _ := right.(uuid.NullUUID).MarshalText()
		return string(leftStr), string(rightStr), true

	case sql.NullString:
		leftStr := typedLeft.String
		if !typedLeft.Valid {
			leftStr = "null"
		}

		rightStr := right.(sql.NullString).String
		if !right.(sql.NullString).Valid {
			rightStr = "null"
		}

		return leftStr, rightStr, true

	case sql.NullInt64:
		var leftInt64Ptr *int64
		var rightInt64Ptr *int64
		if !typedLeft.Valid {
			leftInt64Ptr = nil
		} else {
			leftInt64Ptr = ptr(typedLeft.Int64)
		}

		rightInt64Ptr = ptr(right.(sql.NullInt64).Int64)
		if !right.(sql.NullInt64).Valid {
			rightInt64Ptr = nil
		}

		return leftInt64Ptr, rightInt64Ptr, true
	case database.TemplateACL:
		return fmt.Sprintf("%+v", left), fmt.Sprintf("%+v", right), true
	default:
		return left, right, false
	}
}

// derefPointer deferences a reflect.Value that is a pointer to its underlying
// value. It dereferences recursively until it finds a non-pointer value. If the
// pointer is nil, it will be coerced to the zero value of the underlying type.
func derefPointer(ptr reflect.Value) reflect.Value {
	if !ptr.IsNil() {
		// Grab the value the pointer references.
		ptr = ptr.Elem()
	} else {
		// Coerce nil ptrs to zero'd values of their underlying type.
		ptr = reflect.Zero(ptr.Type().Elem())
	}

	// Recursively deref nested pointers.
	if ptr.Kind() == reflect.Ptr {
		return derefPointer(ptr)
	}

	return ptr
}

func ptr[T any](x T) *T {
	return &x
}
