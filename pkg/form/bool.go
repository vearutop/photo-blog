package form

type Bool3 string

func (Bool3) Enum() []any {
	return []any{"undefined", "yes", "no"}
}

func (b Bool3) True() bool {
	return b == "yes"
}
