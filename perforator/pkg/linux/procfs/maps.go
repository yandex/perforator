package procfs

import (
	"fmt"
)

func parseHexSymb(symb byte) (uint64, bool) {
	if '0' <= symb && symb <= '9' {
		return uint64(symb - '0'), true
	}
	if 'a' <= symb && symb <= 'f' {
		return uint64(symb-'a') + 10, true
	}
	return 0, false
}

func parseDecSymb(symb byte) (uint64, bool) {
	if '0' <= symb && symb <= '9' {
		return uint64(symb - '0'), true
	}
	return 0, false
}

func trimLeftSpaces(line []byte, begin int) []byte {
	i := begin
	for i < len(line) && line[i] == ' ' {
		i++
	}
	return line[i:]
}

func parseMappingPermissions(perms []byte) MappingPermissions {
	flags := MappingPermissionNone
	if perms[0] == 'r' {
		flags |= MappingPermissionReadable
	}
	if perms[1] == 'w' {
		flags |= MappingPermissionWriteable
	}
	if perms[2] == 'x' {
		flags |= MappingPermissionExecutable
	}
	if perms[3] == 'p' {
		flags |= MappingPermissionPrivate
	} else {
		flags |= MappingPermissionShared
	}
	return flags
}

// written for scanning and converting in one time, and not to allocate useless strings
func scanInt(line []byte, begin int, base uint64) (uint64, []byte, error) {
	var res uint64
	if base != 10 && base != 16 {
		return 0, nil, fmt.Errorf("unsupported base %d", base)
	}
	for i := begin; i < len(line); i++ {
		var value uint64
		var isParsed bool
		if base == 16 {
			value, isParsed = parseHexSymb(line[i])
		} else if base == 10 {
			value, isParsed = parseDecSymb(line[i])
		}
		if !isParsed {
			return res, line[i:], nil
		}
		res = res*base + value
	}
	return res, nil, nil
}

func ParseProcessMapping(mapping *Mapping, line []byte, path *string) error {
	var err error
	sourceLine := line
	mapping.Begin, line, err = scanInt(line, 0, 16)
	if err != nil || line[0] != '-' {
		return fmt.Errorf("failed to parse mapping begin in %s line %q", *path, string(sourceLine))
	}

	mapping.End, line, err = scanInt(line, 1, 16)
	if err != nil || line[0] != ' ' {
		return fmt.Errorf("failed to parse mapping end in %s line %q", *path, string(sourceLine))
	}

	perms := line[1:5]
	line = line[5:]
	if line[0] != ' ' {
		return fmt.Errorf("malformed %s line %q", *path, string(line))
	}
	mapping.Permissions = parseMappingPermissions(perms)

	offset, line, err := scanInt(line, 1, 16)
	if err != nil || line[0] != ' ' {
		return fmt.Errorf("failed to parse mapping offset in %s line %q", *path, string(sourceLine))
	}
	mapping.Offset = int64(offset)

	deviceMaj, line, err := scanInt(line, 1, 16)
	if err != nil || line[0] != ':' {
		return fmt.Errorf("failed to parse device maj in %s line %q", *path, string(sourceLine))
	}
	mapping.Device.Maj = uint32(deviceMaj)

	deviceMin, line, err := scanInt(line, 1, 16)
	if err != nil || line[0] != ' ' {
		return fmt.Errorf("failed to parse device min in %s line %q", *path, string(sourceLine))
	}
	mapping.Device.Min = uint32(deviceMin)

	mapping.Inode.ID, line, err = scanInt(line, 1, 10)
	if err != nil || (len(line) > 0 && line[0] != ' ') {
		return fmt.Errorf("failed to parse inode id in %s line %q", *path, string(sourceLine))
	}

	if len(line) > 1 {
		pathBegin := trimLeftSpaces(line, 1)
		mapping.Path = string(pathBegin)
	}
	return nil
}
