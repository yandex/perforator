package procfs

import "errors"

var bootTime uint64

func init() {
	stat, err := FS().GetStat()
	if err != nil {
		return
	}
	bootTime = stat.Btime
}

func GetBootTime() (uint64, error) {
	if bootTime == 0 {
		return 0, errors.New("boot time was not initialized successfully")
	}

	return bootTime, nil
}
