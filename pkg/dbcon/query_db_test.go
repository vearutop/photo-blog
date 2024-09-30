package dbcon_test

import (
	"github.com/stretchr/testify/assert"
	"github.com/vearutop/photo-blog/pkg/dbcon"
	"testing"
)

func TestSplitStatements(t *testing.T) {
	s := `SELECT "ol'o""lo","",'';
	-- next sta;tement
	SELECT * FROM refers order by ts desc limit 15;
	-- next statement
	SELECT * FROM visitor WHERE hash=8361038239347337526;
SELECT /* 1;2;3 */ 'aaa' 
`

	res := dbcon.SplitStatements(s)
	assert.Equal(t, []string{
		`SELECT "ol'o""lo","",''`,
		"-- next sta;tement\n\tSELECT * FROM refers order by ts desc limit 15",
		"-- next statement\n\tSELECT * FROM visitor WHERE hash=8361038239347337526",
		"SELECT /* 1;2;3 */ 'aaa'",
	}, res)
}

func TestSplitStatements2(t *testing.T) {
	s := `";";';'`
	res := dbcon.SplitStatements(s)

	assert.Equal(t, []string{
		`";"`,
		`';'`,
	}, res)
}
