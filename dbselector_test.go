package dbselector

import (
	"reflect"
	"testing"
	"time"
)

func TestSelectorCorrectnessWithoutWhere(t *testing.T) {

	selector := &Selector{}

	selector.Select("user")
	sql, binds := selector.Sql()

	gageSql := "SELECT * FROM \"user\""
	compareSql(t, gageSql, sql)
	compareBinds(t, binds, map[string]interface{}{})
}

func TestSelectorCorrectnessWithWhere(t *testing.T) {

	selector := &Selector{}

	selector.Select("user").Where("name", "=", "Vova")
	sql, binds := selector.Sql()

	gageSql := "SELECT * FROM \"user\" WHERE name = :name1"
	compareSql(t, gageSql, sql)

	gage := map[string]interface{}{"name1": "Vova"}
	compareBinds(t, binds, gage)
}

func TestSelectorCorrectnessWithWhereTrue(t *testing.T) {

	selector := &Selector{}

	// здесь при пустом массиве в WhereIn подставляется кляуза WhereTrue
	selector.Select("user").WhereIn("id", []interface{}{})
	sql, _ := selector.Sql()

	gageSql := "SELECT * FROM \"user\" WHERE true"
	compareSql(t, gageSql, sql)
}

func TestSelectorCorrectnessWithComplexClause(t *testing.T) {

	selector := &Selector{}

	selector.Select("user").OpenBracket().Where("name", "=", "Vova").OpenBracket().
		And("age", ">", 36).Or("age", "<", 18).CloseBracket().CloseBracket().
		AndIn("height", []interface{}{170, 171, 172}).
		Or("active", "=", false).OrderBy("name").Limit(5).Offset(10)
	sql, binds := selector.Sql()

	gageSql := "SELECT * FROM \"user\" WHERE ( name = :name1 " +
		"AND ( age > :age2 OR age < :age3) )  AND height IN (:height4,:height5,:height6) OR active = :active7 " +
		"ORDER BY name LIMIT 5 OFFSET 10"
	compareSql(t, gageSql, sql)

	gage := map[string]interface{}{
		"name1":   "Vova",
		"age2":    36,
		"age3":    18,
		"height4": 170,
		"height5": 171,
		"height6": 172,
		"active7": false,
	}
	compareBinds(t, binds, gage)
}

func TestSelectorWithCount(t *testing.T) {

	sel := &Selector{}
	sel.Select("user").Where("name", "=", "Vova").Count()
	sql, binds := sel.Sql()

	gageSql := "SELECT count(*) FROM \"user\" WHERE name = :name1"
	compareSql(t, gageSql, sql)

	gage := map[string]interface{}{"name1": "Vova"}
	compareBinds(t, binds, gage)
}

func TestSelectorOrderBind(t *testing.T) {

	sel := &Selector{}
	sel.Select("user").OrderBind("email", "ASC").OrderBind("name", "DESC")
	sql, binds := sel.Sql()

	gageSql := "SELECT * FROM \"user\" ORDER BY :email1 asc, :name2 desc"
	compareSql(t, gageSql, sql)

	gage := map[string]interface{}{"email1": "email", "name2": "name"}
	compareBinds(t, binds, gage)
}

func TestSelectorDelete(t *testing.T) {
	sel := &Selector{}
	sel.Delete("user").Where("id", ">", "7")
	sql, binds := sel.Sql()

	gageSql := "DELETE FROM \"user\" WHERE id > :id1"
	compareSql(t, gageSql, sql)

	gage := map[string]interface{}{"id1": "7"}
	compareBinds(t, binds, gage)
}

func TestSelectorUpdate(t *testing.T) {

	sel := &Selector{}
	sel.Update("post").Set("title", "Test").Set("author_id", 4)
	sel.Where("active", "=", true).Or("id", "=", 77)
	sql, binds := sel.Sql()

	gageSql := "UPDATE \"post\" SET title = :title1, author_id = :author_id2" +
		" WHERE active = :active3 OR id = :id4"
	compareSql(t, gageSql, sql)

	gageBind := map[string]interface{}{
		"title1":     "Test",
		"author_id2": 4,
		"active3":    true,
		"id4":        77,
	}
	compareBinds(t, binds, gageBind)
}

func TestSelectorInsertEmptyValue(t *testing.T) {

	sel := &Selector{}

	emptyValue := make([]interface{}, 0)
	sel.Insert("table").Values(emptyValue)
	sql, binds := sel.Sql()

	gageSql := "INSERT INTO \"table\""
	compareSql(t, gageSql, sql)

	gageBind := map[string]interface{}{}
	compareBinds(t, binds, gageBind)
}

type newType int64

type testStruct struct {
	Id    int64
	Num_A int64
	NumB  int64     `db:"num_b" json:"num_b"`
	Time  time.Time `json:"-" db:"time"`
	NumC  newType   `db:"num_c" json:"-"`
	NumD  int64     `db:"-"`
}

func TestSelectorInsertOneItem(t *testing.T) {

	item := testStruct{
		Num_A: 2,
		NumB:  3,
		Time:  time.Now(),
		NumC:  newType(4),
	}

	sel := &Selector{}
	sel.Insert("table").Values([]interface{}{item})
	sql, binds := sel.Sql()

	gageSql := "INSERT INTO \"table\" (Num_A, num_b, time, num_c) VALUES " +
		"(:Num_A1, :num_b2, :time3, :num_c4)"
	compareSql(t, gageSql, sql)

	gageBind := map[string]interface{}{
		"Num_A1": item.Num_A,
		"num_b2": item.NumB,
		"time3":  item.Time,
		"num_c4": item.NumC,
	}
	compareBinds(t, binds, gageBind)
}

func TestSelectorInsert(t *testing.T) {
	item1 := testStruct{
		Num_A: 2,
		NumB:  3,
		Time:  time.Now(),
		NumC:  newType(4),
	}

	item2 := testStruct{
		Num_A: 20,
		NumB:  30,
		Time:  time.Now().Add(24 * time.Hour),
		NumC:  newType(40),
	}

	sel := &Selector{}
	sel.Insert("table").Values([]interface{}{item1, item2})
	sql, binds := sel.Sql()

	gageSql := "INSERT INTO \"table\" (Num_A, num_b, time, num_c) VALUES " +
		"(:Num_A1, :num_b2, :time3, :num_c4), (:Num_A5, :num_b6, :time7, :num_c8)"
	compareSql(t, gageSql, sql)

	gageBind := map[string]interface{}{
		"Num_A1": item1.Num_A,
		"num_b2": item1.NumB,
		"time3":  item1.Time,
		"num_c4": item1.NumC,
		"Num_A5": item2.Num_A,
		"num_b6": item2.NumB,
		"time7":  item2.Time,
		"num_c8": item2.NumC,
	}
	compareBinds(t, binds, gageBind)
}

func TestSelectorInsertReturning(t *testing.T) {
	item := testStruct{
		Num_A: 2,
		NumB:  3,
		Time:  time.Now(),
		NumC:  newType(4),
	}

	sel := &Selector{}
	sel.Insert("table").Values([]interface{}{item})
	sel.Returning("id", "num_b")
	sql, _ := sel.Sql()

	gageSql := "INSERT INTO \"table\" (Num_A, num_b, time, num_c) VALUES " +
		"(:Num_A1, :num_b2, :time3, :num_c4) " +
		"RETURNING id,num_b"
	compareSql(t, gageSql, sql)
}

func TestRepeatingParam(t *testing.T) {

	sel := &Selector{}
	sel.Select("user").Where("id", ">", 10).And("id", "<", 55)
	sql, binds := sel.Sql()

	gageSql := "SELECT * FROM \"user\" WHERE id > :id1 AND id < :id2"
	compareSql(t, gageSql, sql)

	gageBinds := map[string]interface{}{"id1": 10, "id2": 55}
	compareBinds(t, binds, gageBinds)
}

func TestParamPrefix(t *testing.T) {

	sel := &Selector{}
	sel.SetParameterPrefix("q1_")
	sel.Select("table").Where("name", "=", "Vova").OrderBind("id", "DESC")
	sql, binds := sel.Sql()

	gageSql := "SELECT * FROM \"table\" WHERE name = :q1_name1 ORDER BY :q1_id2 desc"
	compareSql(t, gageSql, sql)

	gageBinds := map[string]interface{}{"q1_name1": "Vova", "q1_id2": "id"}
	compareBinds(t, binds, gageBinds)
}

func TestRawQuery(t *testing.T) {

	sel := &Selector{}
	sel.Delete("user").Where("id", ">", 137).Or("name", "LIKE", "%Vov%")
	sql, binds := sel.RawSql()

	gageSql := "DELETE FROM \"user\" WHERE id > $1 OR name LIKE $2"
	compareSql(t, gageSql, sql)

	gageBinds := []interface{}{137, "%Vov%"}
	compareBinds(t, binds, gageBinds)
}

func compareBinds(t *testing.T, binds interface{}, gage interface{}) {
	if !reflect.DeepEqual(binds, gage) {
		t.Errorf("GAGE:  %v\nBINDS: %v\n Элементы в отображениях не совпадают.", gage, binds)
	}
}

func compareSql(t *testing.T, gage string, sql string) {

	if gage != sql {
		t.Errorf("Эталон:\n%v\nВозвращено:\n%v\n", gage, sql)
	}
}
