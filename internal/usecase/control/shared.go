package control

func stripVal[V any](v V, err error) error {
	return err
}

func must[V any](v V, err error) V {
	if err != nil {
		panic(err)
	}

	return v
}
