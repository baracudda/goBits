package sqlBits

import (
	"reflect"
	"strings"
)

// ISqlSanitizer UI defined values like sort order, pager info, and requested
// fields desired can be an attack vector (SQL Injection) if not properly sanitized.
// If your class can protect against such attack vectors, define this interface
// so that a SqlBuilder instance can query your object for sanitization methods.
type ISqlSanitizer interface {
	// GetDefinedFields Returns the array of defined fields available.
	GetDefinedFields() []string
	// IsFieldSortable Returns TRUE if the fieldname specified is sortable.
	IsFieldSortable( aFieldName string ) bool
	// GetDefaultSort Return the default sort definition.
	GetDefaultSort() OrderByList
	// GetSanitizedOrderByList Providing click-able headers in tables to easily sort them
	// by a particular field is a great UI feature. However, in order to prevent SQL injection
	// attacks, we must double-check that a supplied field name to order the query by is
	// something we can sort on; this method makes use of the IsFieldSortable()
	// method to determine if the browser supplied field name is one of our possible
	// headers that can be clicked on for sorting purposes.
	GetSanitizedOrderByList( aList OrderByList ) OrderByList
	// GetSanitizedFieldList Prune the field list to remove any invalid fields.
	GetSanitizedFieldList( aFieldList []string ) []string
}

// IsStructFieldExported IsExported reports whether the struct field is exported.
func IsStructFieldExported( f reflect.StructField ) bool {
	return f.PkgPath == ""
}

// FieldNameTag Custom tag to use for DetermineFieldsFromTableStruct
var FieldNameTag = ""
// DefaultFieldNameStrConvFunc String-conversion func for struct field name to query field name
var DefaultFieldNameStrConvFunc = strings.ToLower

// DetermineFieldsFromTableStruct Returns the array of publicly defined fields available.
func DetermineFieldsFromTableStruct( aTableStruct interface{} ) []string {
	var theResult []string
	rowVal := reflect.ValueOf(aTableStruct)
	rowType := reflect.TypeOf(aTableStruct)
	for i:=0; i<rowType.NumField(); i++ {
		theField := rowType.Field(i)
		if IsStructFieldExported(theField) {
			if rowVal.Field(i).Kind() != reflect.Struct {
				theName := theField.Name
				// see if we have a "sql" tag to use
				theQueryResultName := theField.Tag.Get("sql")
				if theQueryResultName == "" {
					// else see if we have a "db" tag to use
					theQueryResultName = theField.Tag.Get("db")
				}
				if theQueryResultName == "" && FieldNameTag != "" {
					// else see if we have a custom tag to use
					theQueryResultName = theField.Tag.Get(FieldNameTag)
				}
				if theQueryResultName == "" {
					theQueryResultName = DefaultFieldNameStrConvFunc(theName)
				}
				theResult = append(theResult, theQueryResultName)
			} else {
				theEmbeddedFields := DetermineFieldsFromTableStruct(rowVal.Field(i))
				theResult = append(theResult, theEmbeddedFields...)
			}
		}
	}
	return theResult
}

// IsFieldSortable Returns TRUE if the fieldname specified is sortable.
// Set public field tag to `sortable:"false"` if its not sortable.
func IsFieldSortable( aTableStruct interface{}, aFieldName string ) bool {
	theName := strings.ToTitle(aFieldName)
	theField, found := reflect.TypeOf(aTableStruct).FieldByName(theName)
	if found {
		theSortableTag := theField.Tag.Get("sortable")
		return theSortableTag != "false"
	} else {
		return false
	}
}

// GetSanitizedOrderByList Providing click-able headers in tables to easily sort
// them by a particular field is a great UI feature. However, in order to prevent
// SQL injection attacks, we must double-check that a supplied field name to order
// the query by is something we can sort on; this method makes use of the
// IsFieldSortable() method to determine if the browser supplied field name is
// one of our possible headers that can be clicked on for sorting purposes.
func GetSanitizedOrderByList( aTableStruct interface{}, aList OrderByList ) OrderByList {
	sList := OrderByList{}
	for k, v := range aList {
		if IsFieldSortable(aTableStruct, k) {
			sList[k] = v
		}
	}
	return sList
}

// GetSanitizedFieldList Prune the field list to remove any invalid fields.
func GetSanitizedFieldList( aTableStruct interface{}, aFieldList []string ) []string {
	var sList []string
	for _, v := range aFieldList {
		theName := strings.ToTitle(v)
		_, found := reflect.TypeOf(aTableStruct).FieldByName(theName)
		if found {
			sList = append(sList, v)
		}
	}
	return sList
}
