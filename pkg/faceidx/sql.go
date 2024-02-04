package faceidx

import "fmt"

type SQL struct {
}

func CreateFacesTable() string {
	res := `CREATE TABLE faces ( id integer primary key,`
	for i := 0; i < 128; i++ {
		res += fmt.Sprintf(`f%d REAL,`, i)
	}

	return res[0:len(res)-1] + ")"
}

func Insert(descriptor ...[128]float64) string {
	res := "INSERT INTO faces ("
	values := " VALUES "

	for i := 0; i < 128; i++ {
		res += fmt.Sprintf(`f%d,`, i)
	}

	for d := 0; d < len(descriptor); d++ {
		values += "("
		for i := 0; i < 128; i++ {
			values += fmt.Sprintf(`%f,`, descriptor[d][i])
		}

		values = values[0:len(values)-1] + "),"

	}

	res = res[0:len(res)-1] + ")" + values[0:len(values)-1] + ";"

	return res
}

func SelectSimilar(descriptor [128]float64) string {
	res := "SELECT id, sqrt("
	for i := 0; i < 128; i++ {
		res += fmt.Sprintf("pow(%f-f%d,2)+", descriptor[i], i)
	}

	res = res[0:len(res)-1] + ") AS dist FROM faces"

	return res
}

func SelectSimilarID(id int) string {
	res := "SELECT j.id, sqrt("
	for i := 0; i < 128; i++ {
		res += fmt.Sprintf("pow(b.f%d-j.f%d,2)+", i, i)
	}

	res = res[0:len(res)-1] + fmt.Sprintf(") AS dist FROM faces AS j INNER JOIN faces AS b ON b.id = %d", id)

	return res
}
