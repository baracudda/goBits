/*
 * Copyright (C) 2021 Blackmoon Info Tech Services
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package sqlBits

import (
	"regexp"
	"strconv"
	"strings"
)

type DbModeler interface {
	DbMetatater
	DbTransactioner
}

type IDataSource interface {
	IsKeyDefined( aKey string ) bool
	IsKeyValueAList( aKey string ) bool
	GetValueForKey( aKey string ) *string
	GetValueListForKey( aKey string ) *[]string
}

// Use this class to help build SQL queries.
// Supports: MySQL and Postgres.
type Builder struct {
	// Database model used to tweak SQL dialect specifics.
	myDbModel       DbModeler
	// Used to determine if we started a transaction or not.
	// The flag is incremented every time a transaction is requested
	//   and decremented when commited; only begins/commits when transitioning
	//   from 0 to 1 and back to 0. This allows us to "nest" transactions.
	myTransactionFlag int
	// If set, parameter data is retrieved from it.
	myDataSource IDataSource

	/**
	 * The object used to sanitize field/orderby lists to help prevent
	 * SQL injection attacks.
	 * @var ISqlSanitizer
	 */
	//public $mySqlSanitizer = null;

	// The SQL string being built.
	mySql           string
	// SQL statement parameters to use (contains all keys from mySetParams, too).
	myParams        map[string]*string
	// used by SQL() if driver does not support named parameters
	myOrdQuerySql   string
	// used by SQL() if driver does not support named parameters
	myOrdQueryArgs  []interface{}
	// SQL statement set parameters to use.
	mySetParams     map[string]*[]string
	// SQL statement parameter types (not sure if we need them, yet)
	//myParamTypes    map[string]string
	// Prefix for a parameter about to be added.
	myParamPrefix   string
	// Operator for the parameter to use. e.g. " LIKE ", "=", "<>", etc.
	myParamOperator string

	// Using the "=" when NULL is involved is ambiguous unless you know
	// if it is part of a SET clause or WHERE clause.  Explicitly set
	// this flag to let the SqlBuilder know it is in a WHERE clause.
	bUseIsNull bool
}

// Models can use this package to help build their SQL queries.
func NewBuilder( aDbModeler DbModeler ) *Builder {
	return new(Builder).WithModel(aDbModeler)
}

// Initializer like NewBuilder. e.g.: new(Builder).WithModel(aDbModeler)
func (sqlbldr *Builder) WithModel( aDbModeler DbModeler ) *Builder {
	if aDbModeler == nil {
		panic("no DbModeler defined!")
	}
	sqlbldr.myDbModel = aDbModeler
	return sqlbldr.Reset()
}

// Resets the object so it can be resused without creating a new instance.
func (sqlbldr *Builder) Reset() *Builder {
	sqlbldr.mySql = ""
	sqlbldr.myParams = map[string]*string{}
	sqlbldr.mySetParams = map[string]*[]string{}
	//sqlbldr.myParamTypes = map[string]string{}
	sqlbldr.myParamPrefix = " "
	sqlbldr.myParamOperator = "="
	sqlbldr.bUseIsNull = false
	return sqlbldr
}

// If we are not already in a transaction, start one.
func (sqlbldr *Builder) BeginTransaction() *Builder {
	if sqlbldr.myTransactionFlag < 1 {
		if !sqlbldr.myDbModel.InTransaction() {
			sqlbldr.myDbModel.BeginTransaction()
		}
	}
	sqlbldr.myTransactionFlag += 1
	return sqlbldr
}

// If we started a transaction earlier, commit it.
func (sqlbldr *Builder) CommitTransaction() *Builder {
	if sqlbldr.myTransactionFlag > 0 {
		if sqlbldr.myTransactionFlag -= 1; sqlbldr.myTransactionFlag == 0 {
			sqlbldr.myDbModel.CommitTransaction()
		}
	}
	return sqlbldr
}

// If we started a transaction earlier, roll it back.
func (sqlbldr *Builder) RollbackTransaction() *Builder {
	if sqlbldr.myTransactionFlag > 0 {
		if sqlbldr.myTransactionFlag -= 1; sqlbldr.myTransactionFlag == 0 {
			sqlbldr.myDbModel.RollbackTransaction()
		}
	}
	return sqlbldr
}

// Quoted identifiers are DB vendor specific so providing a helper method to just
// return a properly quoted string for MySQL vs MSSQL vs Oracle, etc. is handy.
func (sqlbldr *Builder) GetQuoted( aIdentifier string ) string {
	delim := string(sqlbldr.myDbModel.GetDbMeta().IdentifierDelimiter)
	return delim + strings.Replace(aIdentifier, delim, delim+delim, -1) + delim
}

// Sets the SQL string to this value to build upon.
func (sqlbldr *Builder) StartWith( aSql string ) *Builder {
	sqlbldr.mySql = aSql
	return sqlbldr
}

// Some operators require alternate handling during WHERE clauses
// (e.g. "=" with NULLs). Similar to StartWhereClause(), this method is
// specific to building a filter that consists entirely of a
// partial WHERE clause which will get appended to the main SqlBuilder
// using ApplyFilter().
func (sqlbldr *Builder) StartFilter() *Builder {
	sqlbldr.bUseIsNull = true
	sqlbldr.StartWith("1")
	return sqlbldr.SetParamPrefix(" AND ")
}

// Set our param value source.
func (sqlbldr *Builder) SetDataSource( aDataSource IDataSource ) *Builder {
	sqlbldr.myDataSource = aDataSource
	return sqlbldr
}

// Sets the param value and param type, but does not affect the SQL string.
func (sqlbldr *Builder) SetParam( aParamKey string, aParamValue string ) *Builder {
	s := aParamValue
	return sqlbldr.SetNullableParam(aParamKey, &s)
}

// Sets the param value and param type, but does not affect the SQL string.
func (sqlbldr *Builder) SetNullableParam( aParamKey string, aParamValue *string ) *Builder {
	sqlbldr.myParams[aParamKey] = aParamValue
	//sqlbldr.myParamTypes[aParamKey] = "string"
	return sqlbldr
}

// Sets the param value set, but does not affect the SQL string.
func (sqlbldr *Builder) SetParamSet( aParamKey string, aParamValues *[]string ) *Builder {
	sqlbldr.myParams[aParamKey] = nil
	sqlbldr.mySetParams[aParamKey] = aParamValues
	//sqlbldr.myParamTypes[aParamKey] = "string"
	return sqlbldr
}

// Inquire if the data that will be used for a particular param is a set or not.
func (sqlbldr *Builder) IsParamASet( aParamKey string ) bool {
	_, ok := sqlbldr.myParams[aParamKey]
	if ok {
		//if key exists in myParams, then result is if key also exists in mySetParams
		_, ok := sqlbldr.mySetParams[aParamKey]
		return ok
	} else if sqlbldr.myDataSource != nil {
		//if key non-existant, check myDataSource
		return sqlbldr.myDataSource.IsKeyValueAList(aParamKey)
	}
	//no clue about this param, return false
	return false
}

// Gets the current value of a param that has been added.
func (sqlbldr *Builder) GetParam( aParamKey string ) *string {
	val, ok := sqlbldr.myParams[aParamKey]
	if ok {
		return val
	} else {
		return nil
	}
}

// Gets the current value of a param that has been added.
func (sqlbldr *Builder) GetParamSet( aParamKey string ) *[]string {
	valSet, ok := sqlbldr.mySetParams[aParamKey]
	if ok {
		return valSet
	} else {
		return nil
	}
}

// Some SQL drivers require all query parameters be unique. This poses an issue when multiple
// datakeys with the same name are needed in the query (especially true for MERGE queries). This
// method will check for any existing parameters named aParamKey and will return a new
// name with a number for a suffix to ensure its uniqueness.
func (sqlbldr *Builder) GetUniqueParamKey( aParamKey string ) string {
	i := 1
	theKey := aParamKey
	_, bKeyExists := sqlbldr.myParams[theKey]
	for bKeyExists {
		i += 1
		theKey = aParamKey + strconv.Itoa(i)
		_, bKeyExists = sqlbldr.myParams[theKey]
	}
	return theKey
}

// Some operators require alternate handling during WHERE clauses
// (e.g. "=" with NULLs). This will setParamPrefix(" WHERE ") which will
// apply to the next AddParam.
func (sqlbldr *Builder) StartWhereClause() *Builder {
	sqlbldr.bUseIsNull = true
	return sqlbldr.SetParamPrefix(" WHERE ")
}

// Reset WHERE clause flag.
func (sqlbldr *Builder) EndWhereClause() *Builder {
	sqlbldr.bUseIsNull = false
	return sqlbldr
}

// Adds a string to the SQL prefixed with a space (just in case).
// *DO NOT* use this method to write values gathered from
// user input directly into a query. *ALWAYS* use the
// .AddParam() or similar methods, or pre-sanitize the data
// value before writing it into the query.
func (sqlbldr *Builder) Add( aStr string ) *Builder {
	sqlbldr.mySql += " " + aStr
	return sqlbldr
}

// Sets the "glue" string that gets prepended to all subsequent calls to
// AddParam kinds of methods. Spacing is important here, so add what is needed!
func (sqlbldr *Builder) SetParamPrefix( aStr string ) *Builder {
	sqlbldr.myParamPrefix = aStr
	return sqlbldr
}

// Operator string to use in all subsequent calls to addParam methods.
// "=" is default, " LIKE " is a popular operator as well.
func (sqlbldr *Builder) SetParamOperator( aStr string ) *Builder {
	// "!=" is not standard SQL, but is a common programmer mistake, cnv to "<>"
	aStr = strings.Replace(aStr, "!=", OPERATOR_NOT_EQUAL, -1)
	sqlbldr.myParamOperator = aStr
	return sqlbldr
}

// Retrieve the data that will be used for a particular param.
func (sqlbldr *Builder) getParamValueFromDataSource( aParamKey string ) *Builder {
	if sqlbldr.myDataSource != nil {
		if sqlbldr.myDataSource.IsKeyValueAList(aParamKey) {
			return sqlbldr.SetParamSet(aParamKey, sqlbldr.myDataSource.GetValueListForKey(aParamKey))
		} else {
			return sqlbldr.SetNullableParam(aParamKey, sqlbldr.myDataSource.GetValueForKey(aParamKey))
		}
	}
	return sqlbldr
}

// Set a value for a param when its data value is NULL.
func (sqlbldr *Builder) SetParamValueIfNull( aParamKey string, aNewValue string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && (valSet == nil || len(*valSet) == 0) {
		sqlbldr.SetParam(aParamKey, aNewValue)
	} else if !isSet && sqlbldr.GetParam(aParamKey) == nil {
		sqlbldr.SetParam(aParamKey, aNewValue)
	}
	return sqlbldr
}

// Set a value for a param when its data value is empty(). e.g. null|""|0
func (sqlbldr *Builder) SetParamValueIfEmpty( aParamKey string, aNewValue string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && (valSet == nil || len(*valSet) == 0) {
		sqlbldr.SetParam(aParamKey, aNewValue)
	} else if val := sqlbldr.GetParam(aParamKey) ; !isSet && (val == nil || *val == "" || *val == "0") {
		sqlbldr.SetParam(aParamKey, aNewValue)
	}
	return sqlbldr
}

// Mainly used internally by AddParamIfDefined to determine if data param exists.
func (sqlbldr *Builder) isDataKeyDefined( aDataKey string ) bool {
	if sqlbldr.myDataSource != nil {
		return sqlbldr.myDataSource.IsKeyDefined(aDataKey)
	} else {
		return false
	}
}

// Adds to the SQL string as a set of values; e.g. "(:paramkey_1,:paramkey_2,:paramkey_N)"
// Honors the ParamPrefix and ParamOperator properties.
func (sqlbldr *Builder) addParamAsListForColumn( aColumnName string,
	aParamKey string, aDataValuesList *[]string,
) *Builder {
	if aDataValuesList != nil && len(*aDataValuesList) > 0 {
		sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColumnName)
		sqlbldr.mySql += sqlbldr.myParamOperator + "("
		i := 1
		for _, val := range *aDataValuesList {
			theParamKey := aParamKey + "_" + strconv.Itoa(i)
			i += 1
			sqlbldr.mySql += ":" + theParamKey + ","
			sqlbldr.SetParam(theParamKey, val)
		}
		sqlbldr.mySql = strings.TrimRight(sqlbldr.mySql, ",") + ")"
	}
	return sqlbldr
}

// Internal method to affect SQL statment with a param and its value.
func (sqlbldr *Builder) addingParam( aColName string, aParamKey string ) {
	isSet := sqlbldr.IsParamASet(aParamKey)
	if valSet := sqlbldr.GetParamSet(aParamKey); isSet && valSet != nil && len(*valSet) > 0 {
		saveParamOp := sqlbldr.myParamOperator
		switch strings.TrimSpace(sqlbldr.myParamOperator) {
		case "=":
			sqlbldr.myParamOperator = " IN "
		case OPERATOR_NOT_EQUAL:
			sqlbldr.myParamOperator = " NOT IN "
		}//switch
		sqlbldr.addParamAsListForColumn(aColName, aParamKey, valSet)
		sqlbldr.myParamOperator = saveParamOp
	} else {
		if val := sqlbldr.GetParam(aParamKey); val != nil || !sqlbldr.bUseIsNull {
			sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) +
				sqlbldr.myParamOperator + ":" + aParamKey
		} else {
			switch strings.TrimSpace(sqlbldr.myParamOperator) {
			case "=":
				sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) + " IS NULL"
			case OPERATOR_NOT_EQUAL:
				sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColName) + " IS NOT NULL"
			}//switch
		}
	}
}

// Parameter must go into the SQL string regardless of NULL status of data.
func (sqlbldr *Builder) AppendParam( aParamKey string, aParamValue string ) *Builder {
	sqlbldr.SetParam(aParamKey, aParamValue)
	sqlbldr.addingParam(aParamKey, aParamKey)
	return sqlbldr
}

// Parameter must go into the SQL string regardless of NULL status of data.
func (sqlbldr *Builder) MustAddParam( aParamKey string ) *Builder {
	return sqlbldr.MustAddParamForColumn(aParamKey, aParamKey)
}

// Parameter must go into the SQL string regardless of NULL status of data.
// This is a "shortcut" designed to combine calls to setParamValue, and addParam.
func (sqlbldr *Builder) MustAddParamForColumn( aParamKey string, aColumnName string ) *Builder {
	sqlbldr.getParamValueFromDataSource(aParamKey)
	sqlbldr.addingParam(aColumnName, aParamKey)
	return sqlbldr
}

// Parameter only gets added to the SQL string if data IS NOT NULL.
func (sqlbldr *Builder) AddParamIfDefined( aParamKey string ) *Builder {
	if sqlbldr.isDataKeyDefined(aParamKey) {
		sqlbldr.getParamValueFromDataSource(aParamKey)
		sqlbldr.addingParam(aParamKey, aParamKey)
	}
	return sqlbldr
}

// Parameter only gets added to the SQL string if data IS NOT NULL.
func (sqlbldr *Builder) AddParamForColumnIfDefined( aParamKey string, aColumnName string ) *Builder {
	if sqlbldr.isDataKeyDefined(aParamKey) {
		sqlbldr.getParamValueFromDataSource(aParamKey)
		sqlbldr.addingParam(aColumnName, aParamKey)
	}
	return sqlbldr
}

// Adds the list of fields (columns) to the SQL string.
func (sqlbldr *Builder) AddFieldList( aFieldList *[]string ) *Builder {
	theFieldListStr := sqlbldr.myParamPrefix + "*"
	if aFieldList != nil && len(*aFieldList) > 0 {
		theFieldListStr = sqlbldr.myParamPrefix +
			strings.Join(*aFieldList, ", "+sqlbldr.myParamPrefix)
	}
	return sqlbldr.Add(theFieldListStr)
}

// Return the SQL "LIMIT" expression for our model's database type.
func (sqlbldr *Builder) AddQueryLimit( aLimit int, aOffset int ) *Builder {
	if aLimit > 0 && sqlbldr.myDbModel != nil {
		driverName := sqlbldr.myDbModel.GetDbMeta().Name
		switch driverName {
		case MySQL:
		case PostgreSQL:
		default:
			sqlbldr.Add("LIMIT").Add(strconv.Itoa(aLimit))
			if aOffset > 0 {
				sqlbldr.Add("OFFSET").Add(strconv.Itoa(aOffset))
			}
		}//switch
	}
	return sqlbldr
}

// Sub-query gets added to the SQL string.
func (sqlbldr *Builder) AddSubQueryForColumn( aSubQuery *Builder, aColumnName string ) *Builder {
	saveParamOp := sqlbldr.myParamOperator
	switch strings.TrimSpace(sqlbldr.myParamOperator) {
	case "=":
		sqlbldr.myParamOperator = " IN "
	case OPERATOR_NOT_EQUAL:
		sqlbldr.myParamOperator = " NOT IN "
	}//switch
	sqlbldr.mySql += sqlbldr.myParamPrefix + sqlbldr.GetQuoted(aColumnName) +
		sqlbldr.myParamOperator + "(" + aSubQuery.mySql + ")"
	sqlbldr.myParamOperator = saveParamOp
	//also merge in any params from the sub-query
	for k, v := range aSubQuery.myParams {
		sqlbldr.myParams[k] = v
	}
	for k, v := range aSubQuery.mySetParams {
		sqlbldr.mySetParams[k] = v
	}
	return sqlbldr
}

// Apply an externally defined set of WHERE field clauses and param values
// to our SQL (excludes the "WHERE" keyword).
func (sqlbldr *Builder) ApplyFilter( aFilter *Builder ) *Builder {
	if aFilter != nil {
		if aFilter.mySql != "" {
			sqlbldr.mySql += sqlbldr.myParamPrefix + aFilter.mySql
		}
		//also merge in any params from the sub-query
		for k, v := range aFilter.myParams {
			sqlbldr.myParams[k] = v
		}
		for k, v := range aFilter.mySetParams {
			sqlbldr.mySetParams[k] = v
		}
	}
	return sqlbldr
}

// If sort list is defined and its contents are also contained
// in the non-empty $aFieldList, then apply the sort order as neccessary.
// @see ApplyOrderByList() which this method is an alias of.
func (sqlbldr *Builder) ApplySortList( aSortList *map[string]string ) *Builder {
	return sqlbldr.ApplyOrderByList(aSortList)
}

// If order by list is defined, then apply the sort order as neccessary.
// @param aOrderByList map[string]string - keys are the fields => values are
//   'ASC' or 'DESC'.
func (sqlbldr *Builder) ApplyOrderByList( aOrderByList *map[string]string ) *Builder {
	if aOrderByList != nil && sqlbldr.myDbModel != nil {
		theSortKeyword := "ORDER BY"
		/* in case we find diff keywords later...
		driverName := sqlbldr.myDbModel.GetDbMeta().Name
		switch driverName {
		case MySQL:
		case PostgreSQL:
		default:
			theSortKeyword = "ORDER BY"
		}//switch
		 */
		sqlbldr.Add(theSortKeyword)

		theOrderByList := make([]string, len(*aOrderByList))
		idx := 0
		for k, v := range *aOrderByList {
			theEntry := k + " "
			if strings.ToUpper(strings.TrimSpace(v)) == ORDER_BY_DESCENDING {
				theEntry += ORDER_BY_DESCENDING
			} else {
				theEntry += ORDER_BY_ASCENDING
			}
			theOrderByList[idx] = theEntry
		}
		sqlbldr.Add(strings.Join(theOrderByList, ","))
	}
	return sqlbldr
}

// Replace the currently formed SELECT fields with the param.  If you have nested queries,
// you will need to use the
// "SELECT /* FIELDLIST */ field1, field2, (SELECT blah) AS field3 /&#42 /FIELDLIST &#42/ FROM</pre>
// hints in the SQL.
func (sqlbldr *Builder) ReplaceSelectFieldsWith( aSelectFields *[]string ) *Builder {
	if aSelectFields != nil && len(*aSelectFields) > 0 {
		var re *regexp.Regexp
		//nested queries can mess us up, so check for hints first
		if strings.Index(sqlbldr.mySql, FIELD_LIST_HINT_START) > 0 &&
			strings.Index(sqlbldr.mySql, FIELD_LIST_HINT_END) > 0 {
			re = regexp.MustCompilePOSIX("(?i)SELECT /* FIELDLIST */ .+? /* /FIELDLIST */ FROM")
		} else {
			//we want a "non-greedy" match so that it stops at the first "FROM" it finds: ".+?"
			re = regexp.MustCompilePOSIX("(?i)SELECT .+? FROM")
		}
		sqlbldr.mySql = re.ReplaceAllString(sqlbldr.mySql, strings.Join(*aSelectFields, ", "))
	}
	return sqlbldr
}

//Return our currently built SQL statement.
func (sqlbldr *Builder) GetSQLStatement() string {
	return sqlbldr.mySql
}

//Return our currently built SQL statement.
func (sqlbldr *Builder) SQL() string {
	if sqlbldr.myParams != nil && len(sqlbldr.myParams) > 0 &&
		sqlbldr.myDbModel != nil && !sqlbldr.myDbModel.GetDbMeta().SupportsNamedParams {
		sqlbldr.myOrdQuerySql = sqlbldr.mySql
		i := 1
		for k, v := range sqlbldr.myParams {
			theOldKey := ":"+k
			theNewKey := "$"+strconv.Itoa(i)
			if strings.Contains(sqlbldr.myOrdQuerySql, theOldKey) && v != nil {
				sqlbldr.myOrdQuerySql = strings.Replace(sqlbldr.myOrdQuerySql, theOldKey, theNewKey, 1)
				sqlbldr.myOrdQueryArgs = append(sqlbldr.myOrdQueryArgs, *v)
				i += 1
			}
		}
		return sqlbldr.myOrdQuerySql
	} else {
		return sqlbldr.mySql
	}
}

//Return our current SQL params in use.
func (sqlbldr *Builder) SQLparams() map[string]*string {
	if sqlbldr.myParams != nil {
		return sqlbldr.myParams
	} else {
		return map[string]*string{}
	}
}

//Return our current SQL param sets in use.
func (sqlbldr *Builder) SQLparamSets() map[string]*[]string {
	if sqlbldr.mySetParams != nil {
		return sqlbldr.mySetParams
	} else {
		return map[string]*[]string{}
	}
}

//Return SQL query arguments IFF the driver does not support named parameters.
func (sqlbldr *Builder) SQLargs() []interface{} {
	return sqlbldr.myOrdQueryArgs
}


/*

/**
 * Set the object used for sanitizing SQL to help prevent SQL Injection attacks.
 * @param ISqlSanitizer $aSqlSanitizer - the object used to sanitize field/orderby lists.
 * @return $this Returns $this for chaining.
 * /
public func (sqlbldr *Builder) setSanitizer( ISqlSanitizer $aSanitizer=null )
{
sqlbldr.mySqlSanitizer = $aSanitizer;
return sqlbldr
}

/**
 * Retrieve the order by list from the sanitizer which might be from the UI or a default.
 * @return $this Returns $this for chaining.
 * /
public func (sqlbldr *Builder) applyOrderByListFromSanitizer()
{
if ( !empty(sqlbldr.mySqlSanitizer) )
return sqlbldr.applyOrderByList( sqlbldr.mySqlSanitizer->getSanitizedOrderByList() ) ;
else
return sqlbldr
}

//=================================================================
// MAPPED func (sqlbldr *Builder)S TO MODEL
//=================================================================

/**
 * Execute DML (data manipulation language - INSERT, UPDATE, DELETE) statements.
 * @throws DbException if there is an error.
 * @return number|\PDOStatement Returns the number of rows affected OR if using params,
 *   the PDOStatement.
 * @see \BitsTheater\Model::execDML();
 * /
public func (sqlbldr *Builder) execDML() {
return sqlbldr.myModel->execDML(sqlbldr.mySql, sqlbldr.myParams, sqlbldr.myParamTypes);
}

/**
 * Executes DML statement and then checks the returned SQLSTATE.
 * @param string|array $aSqlState5digitCodes - standard 5 digit codes to check,
 *   defaults to '02000', meaning "no data"; e.g. UPDATE/DELETE failed due
 *   to record defined by WHERE clause returned no data. May be a comma separated
 *   list of codes or an array of codes to check against.
 * @return boolean Returns the result of the SQLSTATE check.
 * @link https://ib-aid.com/download/docs/firebird-language-reference-2.5/fblangref25-appx02-sqlstates.html
 * /
public func (sqlbldr *Builder) execDMLandCheck($aSqlState5digitCodes=array(self::SQLSTATE_NO_DATA)) {
$theExecResult = sqlbldr.execDML();
if (!empty($aSqlState5digitCodes)) {
$theStatesToCheck = null;
if (is_string($aSqlState5digitCodes)) {
$theStatesToCheck = explode(',', $aSqlState5digitCodes);
} else if (is_array($aSqlState5digitCodes)) {
$theStatesToCheck = $aSqlState5digitCodes;
}
if (!empty($theStatesToCheck)) {
$theSqlState = $theExecResult->errorCode();
return (array_search($theSqlState, $theStatesToCheck, true)!==false);
}
}
return !empty($theExecResult);
}

/**
 * Executes DML statement and then checks the returned SQLSTATE.
 * @param string $aSqlState5digitCode - a single standard 5 digit code to check,
 *   defaults to '02000', meaning "no data"; e.g. UPDATE/DELETE failed due
 *   to record defined by WHERE clause returned no data.
 * @return boolean Returns the result of the SQLSTATE check.
 * @see SqlBuilder::execDMLandCheck()
 * @link https://dev.mysql.com/doc/refman/5.6/en/error-messages-server.html
 * @link https://ib-aid.com/download/docs/firebird-language-reference-2.5/fblangref25-appx02-sqlstates.html
 * /
public func (sqlbldr *Builder) execDMLandCheckCode($aSqlState5digitCode=self::SQLSTATE_NO_DATA) {
return (sqlbldr.execDML()->errorCode()==$aSqlState5digitCode);
}

/**
 * Execute Select query, returns PDOStatement.
 * @throws DbException if there is an error.
 * @return \PDOStatement on success.
 * @see \BitsTheater\Model::query();
 * /
public func (sqlbldr *Builder) query() {
return sqlbldr.myModel->query(sqlbldr.mySql, sqlbldr.myParams, sqlbldr.myParamTypes);
}

/**
 * A combination query & fetch a single row, returns null if errored.
 * @see \BitsTheater\Model::getTheRow();
 * /
public func (sqlbldr *Builder) getTheRow() {
return sqlbldr.myModel->getTheRow(sqlbldr.mySql, sqlbldr.myParams, sqlbldr.myParamTypes);
}

/**
 * SQL Params should be ordered array with ? params OR associative array with :label params.
 * @param array $aListOfParamValues - array of arrays of values for the parameters in the SQL statement.
 * @throws DbException if there is an error.
 * @see \BitsTheater\Model::execMultiDML();
 * /
public func (sqlbldr *Builder) execMultiDML($aListOfParamValues) {
return sqlbldr.myModel->execMultiDML(sqlbldr.mySql, $aListOfParamValues, sqlbldr.myParamTypes);
}

/**
 * Perform an INSERT query and return the new Auto-Inc ID field value for it.
 * Params should be ordered array with ? params OR associative array with :label params.
 * @throws DbException if there is an error.
 * @return int Returns the lastInsertId().
 * @see \BitsTheater\Model::addAndGetId();
 * /
public func (sqlbldr *Builder) addAndGetId() {
return sqlbldr.myModel->addAndGetId(sqlbldr.mySql, sqlbldr.myParams, sqlbldr.myParamTypes);
}

/**
 * Execute DML (data manipulation language - INSERT, UPDATE, DELETE) statements
 * and return the params used in the query. Convenience method when using
 * parameterized queries since PDOStatement::execDML() always only returns TRUE.
 * @throws DbException if there is an error.
 * @return string[] Returns the param data.
 * @see \PDOStatement::execute();
 * /
public func (sqlbldr *Builder) execDMLandGetParams()
{
sqlbldr.myModel->execDML(sqlbldr.mySql, sqlbldr.myParams, sqlbldr.myParamTypes);
return sqlbldr.myParams;
}

/**
 * Sometimes we want to aggregate the query somehow rather than return data from it.
 * @param array $aSqlAggragates - (optional) the aggregation list, defaults to array('count(*)'=>'total_rows').
 * @return array Returns the results of the aggregates.
 * /
public func (sqlbldr *Builder) getQueryTotals( $aSqlAggragates=array('count(*)'=>'total_rows') )
{
$theSqlFields = array();
foreach ($aSqlAggragates as $theField => $theName)
array_push($theSqlFields, $theField . ' AS ' . $theName);
$theSelectFields = implode(', ', $theSqlFields);
$sqlTotals = sqlbldr.cloneFrom($this);
try {
return $sqlTotals->replaceSelectFieldsWith($theSelectFields)
->getAggregateResults(array_values($aSqlAggragates))
;
} catch (\PDOException $pdoe)
{ throw $sqlTotals->newDbException(__METHOD__, $pdoe); }
}

/**
 * Execute the currently built SELECT query and retrieve all the aggregates as numbers.
 * @param string[] $aSqlAggragateNames - the aggregate names to retrieve.
 * @return number[] Returns the array of aggregate values.
 * /
public func (sqlbldr *Builder) getAggregateResults( $aSqlAggragateNames=array('total_rows') )
{
$theResults = array();
//sqlbldr.debugLog(__METHOD__.' sql='.$theSql->mySql.' params='.sqlbldr.debugStr($theSql->myParams));
$theRow = sqlbldr.getTheRow();
if (!empty($theRow)) {
foreach ($aSqlAggragateNames as $theName)
{
$theResults[$theName] = $theRow[$theName]+0;
}
}
return $theResults;
}

/**
 * Providing click-able headers in tables to easily sort them by a particular field
 * is a great UI feature. However, in order to prevent SQL injection attacks, we
 * must double-check that a supplied field name to order the query by is something
 * we can sort on; this method makes use of the <code>Scene::isFieldSortable()</code>
 * method to determine if the browser supplied field name is one of our possible
 * headers that can be clicked on for sorting purposes. The Scene's properties called
 * <code>orderby</code> and <code>orderbyrvs</code> are used to determine the result.
 * @param object $aScene - the object, typically a Scene decendant, which is used
 *   to call <code>isFieldSortable()</code> and access the properties
 *   <code>orderby</code> and <code>orderbyrvs</code>.
 * @param array $aDefaultOrderByList - (optional) default to use if no proper
 *   <code>orderby</code> field was defined.
 * @return array Returns the sanitized OrderBy list.
 * @deprecated Please use SqlBuilder::applyOrderByListFromSanitizer()
 * /
public func (sqlbldr *Builder) sanitizeOrderByList($aScene, $aDefaultOrderByList=null)
{
$theOrderByList = $aDefaultOrderByList;
if (!empty($aScene) && !empty($aScene->orderby))
{
//does the object passed in even define our validation method?
$theHeaderLabel = (method_exists($aScene, 'isFieldSortable'))
? $aScene->isFieldSortable($aScene->orderby)
: null
;
//only valid columns we are able to sort on will define a header label
if (!empty($theHeaderLabel))
{
$theSortDirection = null;
if (isset($aScene->orderbyrvs))
{
$theSortDirection = ($aScene->orderbyrvs)
? self::ORDER_BY_DESCENDING
: self::ORDER_BY_ASCENDING
;
}
$theOrderByList = array( $aScene->orderby => $theSortDirection );
}
}
return $theOrderByList;
}

/**
 * If the Sanitizer is using a pager and limiting the query, try to
 * retrieve the overall query total so we can display "page 1 of 20"
 * or equivalent text/widget.<br>
 * NOTE: this method must be called after the SELECT query is defined,
 * but before the OrderBy/Sort and LIMIT clauses are applied to the SQL
 * string.
 * @return $this Returns $this for chaining.
 * /
public func (sqlbldr *Builder) retrieveQueryTotalsForSanitizer()
{
if ( !empty(sqlbldr.mySqlSanitizer) && sqlbldr.mySqlSanitizer->isTotalRowsDesired() )
{
$theCount = sqlbldr.getQueryTotals();
if ( !empty($theCount) ) {
sqlbldr.mySqlSanitizer->setPagerTotalRowCount(
$theCount['total_rows']
);
}
}
return sqlbldr
}

/**
 * If we have a SqlSanitizer defined, retrieve the query limit information
 * from it and add the SQL limit clause to our SQL string.
 * @return $this Returns $this for chaining.
 * /
public func (sqlbldr *Builder) applyQueryLimitFromSanitizer()
{
if ( !empty(sqlbldr.mySqlSanitizer) )
return sqlbldr.addQueryLimit(
sqlbldr.mySqlSanitizer->getPagerPageSize(),
sqlbldr.mySqlSanitizer->getPagerQueryOffset()
) ;
else
return sqlbldr
}

*/
