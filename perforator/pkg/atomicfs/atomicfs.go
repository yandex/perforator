package atomicfs

// Atomic version of `os.WriteFile`.
func WriteFile(path string, data []byte, opts ...FileOption) error {
	f, err := Create(path, opts...)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Discard()
	}()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return f.Close()
}
