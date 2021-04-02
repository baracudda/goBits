package sqlBits

import (
	"reflect"
	"strings"
)

// UI defined values like sort order, pager info, and requested fields desired
// can be an attack vector (SQL Injection) if not properly sanitized.
// If your class can protect against such attack vectors, define this interface
// so that a SqlBuilder instance can query your object for sanitization methods.
type ISqlSanitizer interface {
	// Returns the array of defined fields available.
	GetDefinedFields() []string
	// Returns TRUE if the fieldname specified is sortable.
	IsFieldSortable( aFieldName string ) bool
	// Return the default sort definition.
	GetDefaultSort() OrderByList
	// Providing click-able headers in tables to easily sort them by a particular field
	// is a great UI feature. However, in order to prevent SQL injection attacks, we
	// must double-check that a supplied field name to order the query by is something
	// we can sort on; this method makes use of the IsFieldSortable()
	// method to determine if the browser supplied field name is one of our possible
	// headers that can be clicked on for sorting purposes.
	GetSanitizedOrderByList( aList OrderByList ) OrderByList
	// Prune the field list to remove any invalid fields.
	GetSanitizedFieldList( aFieldList []string ) []string
}

// Returns the array of publicly defined fields available.
func DetermineFieldsFromTableStruct( aTableStruct interface{} ) []string {
	var theResult []string
	rowType := reflect.TypeOf(aTableStruct)
	for i:=0; i<rowType.NumField(); i++ {
		theName := rowType.Field(i).Name
		lName := strings.ToLower(theName)
		//only return "public" fields, "Id!=id" is public, "id==id" is not.
		if theName != lName {
			theResult = append(theResult, lName)
		}
	}
	return theResult
}

// Returns TRUE if the fieldname specified is sortable.
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

// Providing click-able headers in tables to easily sort them by a particular field
// is a great UI feature. However, in order to prevent SQL injection attacks, we
// must double-check that a supplied field name to order the query by is something
// we can sort on; this method makes use of the IsFieldSortable()
// method to determine if the browser supplied field name is one of our possible
// headers that can be clicked on for sorting purposes.
func GetSanitizedOrderByList( aTableStruct interface{}, aList OrderByList ) OrderByList {
	sList := OrderByList{}
	for k, v := range aList {
		if IsFieldSortable(aTableStruct, k) {
			sList[k] = v
		}
	}
	return sList
}

// Prune the field list to remove any invalid fields.
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
