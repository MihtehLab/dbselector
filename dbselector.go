package dbselector

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

//Basic type that describes the condition in section WHERE of SQL-query
type clause struct {
	field     string
	operation string
	bind      interface{}
}

// IN-clause
type inClause struct {
	field string
	binds []interface{}
}

// True-clause - used in WhereIn
type trueClause struct{}

type order struct {
	field string
	dir   string
}

type setItem struct {
	field string
	bind  interface{}
}

type (
	whereClause     clause
	andClause       clause
	orClause        clause
	whereInClause   inClause
	andInClause     inClause
	orInClause      inClause
	whereTrueClause trueClause

	bracket bool // true - open bracket, false - close bracket
)

type SqlDialect int

const (
	DIALECT_POSTGRESS = iota
	DIALECT_MYSQL
	DIALECT_SQLITE
)

type SqlQueryType string

const (
	QUERY_SELECT = "SELECT"
	QUERY_DELETE = "DELETE"
	QUERY_UPDATE = "UPDATE"
	QUERY_INSERT = "INSERT"
)

/*Selector - формирует sql-запрос и карту данных для параметризации по заданным условиям.
Смотрите описание методов ниже.
Примеры:
	selector := &Selector{}
	selector.Select("user").Where("name","=","Дима").Or("email","LIKE","%fulleren.io")
	selector.OrderBy("name DESC").Limit(5)
	sql, binds := selector.Sql()
Замечание: при задании условий вызов Where должен идти в первую очередь, т.е.
можно написать selector.Where(...).Or(...).And(...)
но нельзя писать так selector.Or(...).Where(...).And(...). Возможно в следующей версии это
ограничение будет снято.
*/
type Selector struct {
	operation        SqlQueryType  //операция SELECT, DELETE, UPDATE
	tableName        string        //имя таблицы
	orderBy          string        //порядок сортировки
	orders           []order       //список полей для сортировки
	limit            int           //максимальное количество записей, возвращаемых запросом
	offset           int           //смещение при выборке результатов
	count            bool          //указание на подсчет количества элементов в результате
	clauses          []interface{} //список правил для секции WHERE
	parameterPrefix  string        //префикс для названий подставляемых параметров
	parameterCounter int           //счетчик обработанных параметров
	returning        string        //имена полей, возвращаемых при INSERT через запятую
	values           []interface{} //структуры данных для INSERT запроса
	sets             []setItem
	dialect          SqlDialect
}

//Устанавливает префикс для имен подставлемых в запрос параметров
func (s *Selector) SetParameterPrefix(prefix string) {
	s.parameterPrefix = prefix
}

/*Служит для указания имени таблицы к которой производится запрос
Параметры:
	tableName - имя таблицы
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Select("user")
*/
func (s *Selector) Select(tableName string) *Selector {
	s.tableName = tableName
	s.operation = QUERY_SELECT
	return s
}

func (s *Selector) Delete(tableName string) *Selector {
	s.tableName = tableName
	s.operation = QUERY_DELETE
	return s
}

func (s *Selector) Update(tableName string) *Selector {
	s.tableName = tableName
	s.operation = QUERY_UPDATE
	return s
}

func (s *Selector) Insert(tableName string) *Selector {
	s.tableName = tableName
	s.operation = QUERY_INSERT
	return s
}

/* Добавляет к sql запросу WHERE _условие_
Параметры:
	field - имя поля в таблице БД
	operation - опреация используемая для срвнения =, <, >, LIKE и т.п.
	bind - данные для подстановки
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Where("active","=","true")
*/
func (s *Selector) Where(field string, operation string, bind interface{}) *Selector {
	s.clauses = append(s.clauses, whereClause{field, operation, bind})
	return s
}

/*Добавляет к sql запросу условие WHERE поле IN массив
Параметры:
	field - имя поля в таблице БД
	binds - данные для подстановки как массив интерфейсов
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.WhereIn("age", []interface{}{18,19,20,38,39,40})
*/

func (s *Selector) WhereIn(field string, binds []interface{}) *Selector {
	if len(binds) > 0 {
		s.clauses = append(s.clauses, whereInClause{field, binds})
	} else {
		s.clauses = append(s.clauses, whereTrueClause{})
	}
	return s
}

/*Добавляет к sql запросу AND _условие_
Параметры:
	field - имя поля в таблице БД
	operation - опреация используемая для срвнения =, <, >, LIKE и т.п.
	bind - данные для подстановки
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Where("id", ">", 10)
	selector.And("active", "=", "true")
*/
func (s *Selector) And(field string, operation string, bind interface{}) *Selector {
	s.clauses = append(s.clauses, andClause{field, operation, bind})
	return s
}

/*Добавляет к sql запросу условие AND поле IN массив
Параметры:
	field - имя поля в таблице БД
	binds - данные для подстановки как массив интерфейсов
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Where("id", ">", 10)
	selector.AndIn("age", []interface{}{18,19,20,38,39,40})
*/
func (s *Selector) AndIn(field string, binds []interface{}) *Selector {
	if len(binds) > 0 {
		s.clauses = append(s.clauses, andInClause{field, binds})
	}
	return s
}

/*Добавляет к sql запросу OR _условие_
Параметры:
	field - имя поля в таблице БД
	operation - опреация используемая для срвнения =, <, >, LIKE и т.п.
	bind - данные для подстановки
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Where("id", ">", 10)
	selector.Or("active", "=", "true")
*/
func (s *Selector) Or(field string, operation string, bind interface{}) *Selector {
	s.clauses = append(s.clauses, orClause{field, operation, bind})
	return s
}

/*Добавляет к sql запросу условие OR поле IN массив
Параметры:
	field - имя поля в таблице БД
	binds - данные для подстановки как массив интерфейсов
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Where("id", ">", 10)
	selector.OrIn("age", []interface{}{18,19,20,38,39,40})
*/
func (s *Selector) OrIn(field string, binds []interface{}) *Selector {
	if len(binds) > 0 {
		s.clauses = append(s.clauses, orInClause{field, binds})
	}
	return s
}

/*Добавляет к sql запросу открывающую скобку в следующей кляузе
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	...
	selector.OpenBracket()
	selector.Where(...)
*/
func (s *Selector) OpenBracket() *Selector {
	s.clauses = append(s.clauses, bracket(true))
	return s
}

/*Добавляет к sql запросу закрывающую скобку
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	...
	selector.OpenBracket()
	selector.Where(...)
	...
	selector.CloseBracket()
*/
func (s *Selector) CloseBracket() *Selector {
	s.clauses = append(s.clauses, bracket(false))
	return s
}

/*Добавляет к sql запросу UPDATE секцию SET
Параметры:
	field - имя поля в таблице БД
	bind - данные для подстановки
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Update("user").Set("name","Вася").Where("active","=",false)
*/
func (s *Selector) Set(field string, bind interface{}) *Selector {
	si := setItem{field: field, bind: bind}
	s.sets = append(s.sets, si)
	return s
}

/*Задает порядок сортировки
Параметры:
	order - строка вставляемая в секцию ORDER BY
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.OrderBy("name ASC, regdate DESC")
*/
func (s *Selector) OrderBy(order string) *Selector {
	s.orderBy = order
	return s
}

/*Задает порядок сортировки с безопасной подстановкой имен полей в секцию ORDER BY.
Если сортировка задана функцией selector.OrderBy("order_string"), то сортировка будет идти по
строке order_string с игнорированием параметров переданных через selector.OrderBind()
Параметры:
	field - имя поля для сортировки
	dir - направление сортировки. Допустимы значения ASC и DESC, если передано другое то
			будет использовано ASC
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.OrderBind("email","ASC").OrderBind("email","DESC")
*/
func (s *Selector) OrderBind(field string, dir string) *Selector {
	dir = strings.ToLower(dir)
	if dir != "asc" && dir != "desc" {
		dir = "asc"
	}

	s.orders = append(s.orders, order{field: field, dir: dir})

	return s
}

/*Задает имена полей, возвращаемых запросом INSERT
Параметры:
	fields - значение, вставляемое в секцию RETURNING
	это должны быть имена полей через запятую
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Insert("user")
	...
	selector.Returning("id,name") // id - поле таблицы "user"

*/
func (s *Selector) Returning(fields ...string) *Selector {
	for i, field := range fields {
		s.returning += field
		if i < len(fields)-1 {
			s.returning += ","
		}
	}
	return s
}

/*Задает максимальное число записей, возвращаемых запросом
Параметры:
	limit - значение вставляемое в секцию LIMIT
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Limit(20)
*/
func (s *Selector) Limit(limit int) *Selector {
	s.limit = limit
	return s
}

/*Задает какое количество строк в начале нужно пропустить
Параметры:
	OFFSET - значение вставляемое в секцию OFFSET
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Offset(20)
*/
func (s *Selector) Offset(offset int) *Selector {
	s.offset = offset
	return s
}

/*
	Сообщает селектору, что sql-запрос должен возвратить
	количество найденный элементов.
*/
func (s *Selector) Count() *Selector {
	s.count = true
	return s
}

/* Добавляет к sql запросу VALUES _значения_
Параметры:
	data - данные для подстановки
Результат:
	ссылка Selector на самого себя
Пример использования:
	selector := &Selector{}
	selector.Insert("user").Values(users)
*/
func (s *Selector) Values(data []interface{}) *Selector {
	for _, element := range data {
		s.values = append(s.values, element)
	}
	return s
}

/* Формирует параметризированный SQL-запрос и словарь данных для параметризации
Результат:
	1. строка с sql-запросом
	2. словарь с данными для параметризации
Пример использования:
	selector := &Selector{}
	selector.Select("user").Where("name","=","Вася").Or("age","<","18").OrderBy("name DESC").Limit(5)
	sql, binds := selector.Sql()
	тогда sql содержит: SELECT * FROM "user" WHERE name = :name OR age < :age ORDER BY name DESC LIMIT 5
	binds содержит: {"name":"Вася","age":"18"}
*/
func (s *Selector) Sql() (string, map[string]interface{}) {
	return s.sql(false)
}

func (s *Selector) RawSql() (string, []interface{}) {
	sql, binds := s.sql(true)

	resultBinds := make([]interface{}, 0, len(binds))
	for i := 1; i <= len(binds); i++ {
		key := fmt.Sprintf("$%d", i)
		if val, ok := binds[key]; ok {
			resultBinds = append(resultBinds, val)
		}
	}

	return sql, resultBinds
}

func (s *Selector) sql(raw bool) (string, map[string]interface{}) {
	s.parameterCounter = 0
	switch s.operation {
	case QUERY_SELECT:
		return s.selectSql(raw)
	case QUERY_DELETE:
		return s.deleteSql(raw)
	case QUERY_UPDATE:
		return s.updateSql(raw)
	case QUERY_INSERT:
		return s.insertSql(raw)
	default:
		return s.selectSql(raw)
	}
}

//служебный метод возвращающий имя параметра для подстановки
func (s *Selector) getBindingName(param string, raw bool) string {
	s.parameterCounter++
	if !raw {
		return fmt.Sprintf("%v%v%d", s.parameterPrefix, param, s.parameterCounter)
	}

	return fmt.Sprintf("$%d", s.parameterCounter)
}

//возвращает заместитель для псевдонимов в sql-запросе
func (s *Selector) getPlaceholder(bindName string, raw bool) string {
	if !raw {
		return ":" + bindName
	}

	if s.dialect == DIALECT_POSTGRESS {
		return bindName
	}

	return "?"
}

// служебный метод возвращающий имена параметров для подстановки для сравнения IN
// param - имя поля, count - количество элементов в массиве
func (s *Selector) getBindingNamesIN(param string, raw bool, count int) []string {
	// дальше простые проверки на правильности аргумента param
	if count < 1 {
		return []string{}
	}

	var res []string
	for i := 0; i < count; i++ {
		s.parameterCounter++
		if !raw {
			res = append(res, fmt.Sprintf("%v%v%d", s.parameterPrefix, param, s.parameterCounter))
		} else {
			res = append(res, fmt.Sprintf("$%d", s.parameterCounter))
		}
	}

	return res
}

// возвращает массив с заместителями для псевдонимов в sql-запросе для подстановки для сравнения IN
// в качестве bindNames передаётся срез строк, возвращаемый функцией getBindingNamesIN
func (s *Selector) getPlaceholdersIN(bindNames []string, raw bool) string {
	var res string

	res = "("

	for i, p := range bindNames {
		if !raw {
			res += fmt.Sprintf(":%s", p)
		} else if s.dialect == DIALECT_POSTGRESS {
			res += p
		} else {
			res += "?"
		}

		if i < len(bindNames)-1 {
			res += ","
		}
	}

	res += ")"
	return res
}

//формирует запрос вида DELETE FROM table WHERE ...
func (s *Selector) deleteSql(raw bool) (string, map[string]interface{}) {
	resultSql := fmt.Sprintf("DELETE FROM \"%v\"", s.tableName)
	whereSql, binds := s.whereSql(raw)
	resultSql += whereSql

	if s.returning != "" {
		resultSql += " RETURNING " + s.returning
	}

	return resultSql, binds
}

//формирует запрос UPDATE
func (s *Selector) updateSql(raw bool) (string, map[string]interface{}) {
	resultSql := fmt.Sprintf("UPDATE \"%v\" SET", s.tableName)
	binds := map[string]interface{}{}

	for i, si := range s.sets {
		bindName := s.getBindingName(si.field, raw)
		ph := s.getPlaceholder(bindName, raw)
		if i != 0 {
			resultSql += ","
		}
		resultSql += fmt.Sprintf(" %v = %v", si.field, ph)
		binds[bindName] = si.bind
	}

	whereSql, whereBind := s.whereSql(raw)
	resultSql += whereSql

	if s.returning != "" {
		resultSql += " RETURNING " + s.returning
	}

	for k, v := range whereBind {
		binds[k] = v
	}

	return resultSql, binds
}

//формирует запрос типа INSERT INTO ... VALUES ...
func (s *Selector) insertSql(raw bool) (string, map[string]interface{}) {
	resultSQL := fmt.Sprintf("INSERT INTO \"%s\"", s.tableName)
	valuesSql, binds := s.valuesSql(raw)
	resultSQL += valuesSql

	if s.returning != "" {
		resultSQL += " RETURNING " + s.returning
	}

	return resultSQL, binds
}

//формирует values секцию для запроса INSERT и биндинг
func (s *Selector) valuesSql(raw bool) (string, map[string]interface{}) {
	binds := make(map[string]interface{})
	resultSQL := ""
	if len(s.values) == 0 {
		return resultSQL, binds
	}

	// сначала нужно получить имена полей
	fieldNames, fieldNumbers, err := s.getStructFieldNamesForDb(s.values[0])
	if err != nil {
		fmt.Printf("### Error #1 in Selector.valuesSql: %v\n", err)
		return resultSQL, binds
	}

	resultSQL += " ("
	for i, field := range fieldNames {
		if strings.ToLower(field) == "id" {
			// пропуск столбца id
			continue
		}
		resultSQL += field
		if i < len(fieldNames)-1 {
			resultSQL += ", "
		}
	}
	resultSQL += ") VALUES "

	// теперь нужно получить значения полей
	for i, object := range s.values {
		structValues, err := s.getStructFieldValues(object, fieldNumbers)
		if err != nil {
			fmt.Printf("### Error #2 in Selector.valuesSql: %v\n", err)
			return "", binds
		}
		resultSQL += "("
		for j, val := range structValues {
			if strings.ToLower(fieldNames[j]) == "id" {
				// пропуск столбца id
				continue
			}

			bindName := s.getBindingName(fieldNames[j], raw)
			ph := s.getPlaceholder(bindName, raw)
			resultSQL += fmt.Sprintf("%v", ph)
			binds[bindName] = val

			if j < len(structValues)-1 {
				resultSQL += ", "
			}
		}
		resultSQL += ")"
		if i < len(s.values)-1 {
			resultSQL += ", "
		}
	}

	return resultSQL, binds
}

/*
Получает значения полей структуры
Параметры:
structure - структура данных
fieldNumbers - срез номеров полей структуры
Возвращает:
[]interface{} - срез начений полей или пустой срез
error - ошибка или nil
*/
func (sel *Selector) getStructFieldValues(structure interface{}, fieldNumbers []int) ([]interface{}, error) {
	var res []interface{}
	s := reflect.ValueOf(structure)

	for _, fieldNumber := range fieldNumbers {
		if s.Field(fieldNumber).CanInterface() {
			field := s.Field(fieldNumber).Interface()
			res = append(res, field)
		} else {
			return make([]interface{}, 0),
				errors.New("getStructFieldValues: Ошибка преобразования элемента в интерфейс")
		}

	}
	return res, nil
}

/* получает отображение имён полей БД по тегу db: структуры или по имени, в значения
использование имени в качестве ключа происходит если тег db: не указан
structure - структура данных
Возвращает:
[]string - срез имён полей для БД или пустой срез
[]int - номера этих полей в структуре или пустой срез
error - ошибка или nil
*/
func (sel *Selector) getStructFieldNamesForDb(structure interface{}) ([]string, []int, error) {
	s := reflect.ValueOf(structure)
	fields := make([]string, 0)
	fieldNumbers := make([]int, 0)
	var err error

	sType := s.Type()
	for i := 0; i < s.NumField(); i++ { // i это номер поля структуры
		fieldName := sType.Field(i).Name // имя поля структуры
		value := fieldName

		field, ok := reflect.TypeOf(structure).FieldByName(fieldName)
		if !ok {
			err = errors.New("reflect: Поле структуры не найдено!")
			return make([]string, 0), make([]int, 0), err
		}

		tagString := string(field.Tag)
		keyIndex := strings.Index(tagString, "db:") // откуда начинаются значения для ключа db:
		if keyIndex > -1 {                          // иначе value уже равно fieldName
			tagString = tagString[keyIndex:] // отбрасываем то, что в строке до найденного ключа
			// теперь ищем пару кавычек
			q1Index := strings.Index(tagString, "\"") // индекс открывающей кавычки
			if q1Index == -1 {
				err = errors.New("Отсутствует открывающая кавычка")
				return make([]string, 0), make([]int, 0), err
			}
			qString := tagString[q1Index:]                  // qString теперь равно строке начиная с открывающей кавычки
			q2Index := strings.Index(qString[1:], "\"") + 1 // индекс закрывающей кавычки (минуем открывающую кавычку и увеличиваем индекс)
			if q2Index == -1 {
				err = errors.New("Отсутствует закрывающая кавычка")
				return make([]string, 0), make([]int, 0), err
			}
			value = qString[1:q2Index] // то, что между кавычками
			// теперь value содержит значение ключа "db"

			if value == "-" { // пропускаем такое поле
				continue
			}
		}
		fields = append(fields, value)
		fieldNumbers = append(fieldNumbers, i)
	}
	return fields, fieldNumbers, nil
}

//формирует запрос типа SELECT * WHERE ...
func (s *Selector) selectSql(raw bool) (string, map[string]interface{}) {
	selection := "*"
	if s.count {
		selection = "count(*)"
	}
	resultSQL := fmt.Sprintf("SELECT %s FROM \"%s\"", selection, s.tableName)
	whereSql, binds := s.whereSql(raw)
	resultSQL += whereSql

	if s.orderBy != "" {
		resultSQL += s.OrderBySql()
	} else if len(s.orders) > 0 {
		resultSQL += " ORDER BY "
		for i, o := range s.orders {
			bindName := s.getBindingName(o.field, raw)
			ph := s.getPlaceholder(bindName, raw)
			resultSQL += fmt.Sprintf("%v %v", ph, o.dir)
			binds[bindName] = o.field
			if i < len(s.orders)-1 {
				resultSQL += ", "
			}
		}
	}

	resultSQL += s.LimitSql()
	resultSQL += s.OffsetSql()

	return resultSQL, binds
}

//формирует where секцию для запроса и биндинг
func (s *Selector) whereSql(raw bool) (string, map[string]interface{}) {
	binds := make(map[string]interface{})
	resultSQL := ""
	openBrackets := "" // часть строки, содержащая открывающие скобки
	for _, cls := range s.clauses {
		switch cls.(type) {
		case bracket:
			b := bool(cls.(bracket))
			if !b {
				resultSQL += ") "
			} else {
				openBrackets += " ("
			}
		case whereClause:
			wc := cls.(whereClause)
			bindName := s.getBindingName(wc.field, raw)
			ph := s.getPlaceholder(bindName, raw)
			resultSQL += fmt.Sprintf(" WHERE%s %v %v %v", openBrackets, wc.field, wc.operation, ph)
			openBrackets = ""
			binds[bindName] = wc.bind
		case whereInClause:
			wc := cls.(whereInClause)
			bindNames := s.getBindingNamesIN(wc.field, raw, len(wc.binds))
			ph := s.getPlaceholdersIN(bindNames, raw)
			resultSQL += fmt.Sprintf(" WHERE%s %v %s %v", openBrackets, wc.field, "IN", ph)
			openBrackets = ""
			for i, _ := range wc.binds {
				binds[bindNames[i]] = wc.binds[i]
			}
		case whereTrueClause:
			resultSQL += fmt.Sprintf(" WHERE%s true", openBrackets)
			openBrackets = ""
		case andClause:
			ac := cls.(andClause)
			bindName := s.getBindingName(ac.field, raw)
			ph := s.getPlaceholder(bindName, raw)
			resultSQL += fmt.Sprintf(" AND%s %v %v %v", openBrackets, ac.field, ac.operation, ph)
			openBrackets = ""
			binds[bindName] = ac.bind
		case andInClause:
			ac := cls.(andInClause)
			bindNames := s.getBindingNamesIN(ac.field, raw, len(ac.binds))
			ph := s.getPlaceholdersIN(bindNames, raw)
			resultSQL += fmt.Sprintf(" AND%s %v %s %v", openBrackets, ac.field, "IN", ph)
			openBrackets = ""
			for i, _ := range ac.binds {
				binds[bindNames[i]] = ac.binds[i]
			}
		case orClause:
			oc := cls.(orClause)
			bindName := s.getBindingName(oc.field, raw)
			ph := s.getPlaceholder(bindName, raw)
			resultSQL += fmt.Sprintf(" OR%s %v %v %v", openBrackets, oc.field, oc.operation, ph)
			openBrackets = ""
			binds[bindName] = oc.bind
		case orInClause:
			oc := cls.(orInClause)
			bindNames := s.getBindingNamesIN(oc.field, raw, len(oc.binds))
			ph := s.getPlaceholdersIN(bindNames, raw)
			resultSQL += fmt.Sprintf(" OR%s %v %s %v", openBrackets, oc.field, "IN", ph)
			openBrackets = ""
			for i, _ := range oc.binds {
				binds[bindNames[i]] = oc.binds[i]
			}
		}
	}

	return resultSQL, binds
}

/*
	Возвращает секцию LIMIT запроса
*/
func (s *Selector) LimitSql() string {
	if s.limit > 0 {
		return fmt.Sprintf(" LIMIT %d", s.limit)
	}
	return ""
}

/*
	Возвращает секцию OFFSET запроса
*/
func (s *Selector) OffsetSql() string {
	if s.offset > 0 {
		return fmt.Sprintf(" OFFSET %d", s.offset)
	}
	return ""
}

/*
	Возвращает секцию ORDER BY запроса, если параметры
	переданы посредством функции OrderBy (не работает с OrderBind)
*/
func (s *Selector) OrderBySql() string {
	if s.orderBy != "" {
		return fmt.Sprintf(" ORDER BY %s", s.orderBy)
	}
	return ""
}

/*
	Возвращает секцию WHERE запроса и биндинг
*/
func (s *Selector) WhereSql() (string, map[string]interface{}) {
	sql, binds := s.whereSql(false)
	if len(sql) > 6 {
		return sql[6:], binds
	}
	return "", binds
}
