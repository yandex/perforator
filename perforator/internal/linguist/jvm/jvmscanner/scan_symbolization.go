package jvmscanner

import (
	"context"
	"fmt"

	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmbindings"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

type symTreeNodeData struct {
	parentOffset uint32
	methodName   uint32
	// TODO: there is also bci (aka bytecode index), this is needed for line information
}

func (s *Scanner) parseSymbolizationNodeData(nmeta *jvmbindings.NmethodInfo, offset int, cfg config) (symTreeNodeData, error) {
	if offset >= len(nmeta.ScopesData) {
		return symTreeNodeData{}, fmt.Errorf("offset out of range: %d", offset)
	}
	po, nread, err := u5decode(cfg, nmeta.ScopesData[offset:])
	if err != nil {
		return symTreeNodeData{}, fmt.Errorf("failed to read parent offset: %w", err)
	}
	if offset+nread >= len(nmeta.ScopesData) {
		return symTreeNodeData{}, fmt.Errorf("end-of-stream after reading parent offset")
	}
	mID, _, err := u5decode(cfg, nmeta.ScopesData[offset+nread:])
	if err != nil {
		return symTreeNodeData{}, fmt.Errorf("failed to read method ID: %w", err)
	}
	return symTreeNodeData{
		parentOffset: po,
		methodName:   mID,
	}, nil
}

func (s *Scanner) parseInstructionInfo(ctx context.Context, jvm *jvmbindings.JVM, nmethod jvmbindings.CodeBlob) (*jvmsupp.MethodSymbolizationTable, error) {
	nmeta, err := nmethod.NmethodInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get nmethod metadata: %w", err)
	}
	var u5cfg config
	if jvm.Version() >= 20 {
		// TODO: for [20.0.0, 20.0.15) we probably need 19m as well
		u5cfg = u5config20p
	} else {
		u5cfg = u5config19m
	}
	table := new(jvmsupp.MethodSymbolizationTable)

	knownNodes := make(map[uint32]int)
	var nodeCounter int

	methodCounter := 0
	knownMethods := make(map[uint32]int)

	var visit func(uint32) (int, error)

	visit = func(offset uint32) (int, error) {
		nodeID, known := knownNodes[offset]
		if known {
			return nodeID, nil
		}
		if nodeCounter != len(table.Parent) {
			panic(fmt.Sprintf("bug: inconsistent parent array: %d vs %d", nodeCounter, len(table.Parent)))
		}
		if nodeCounter != len(table.Method) {
			panic(fmt.Sprintf("bug: inconsistent method array: %d vs %d", nodeCounter, len(table.Method)))
		}
		nodeID = nodeCounter
		nodeCounter++
		knownNodes[offset] = nodeID
		nodeData, err := s.parseSymbolizationNodeData(&nmeta, int(offset), u5cfg)
		if err != nil {
			return 0, fmt.Errorf("failed to parse symbolization node data at offset %d: %w", offset, err)
		}

		methodID, methodKnown := knownMethods[nodeData.methodName]
		if !methodKnown {
			methodID = methodCounter
			methodCounter++
			knownMethods[nodeData.methodName] = methodID
		}
		table.Method = append(table.Method, uint32(methodID))
		if nodeData.parentOffset != 0 {
			table.Parent = append(table.Parent, ^uint32(0))
			parentID, err := visit(nodeData.parentOffset)
			if err != nil {
				return 0, fmt.Errorf("failed to visit parent node at offset %d: %w", nodeData.parentOffset, err)
			}
			table.Parent[nodeID] = uint32(parentID)
		} else {
			table.Parent = append(table.Parent, uint32(nodeID))
		}
		return nodeID, nil
	}

	var lastOffset uint64
	for i, pcd := range nmeta.PCDs {
		if i == 0 {
			continue
		}
		if pcd.PCOffset < 0 {
			return nil, fmt.Errorf("negative pc offset for PCDesc %d: %d", i, pcd.PCOffset)
		}
		if uint64(pcd.PCOffset) < lastOffset {
			return nil, fmt.Errorf("unexpected non-monotone pc offset %d: %d", i, pcd.PCOffset)
		}
		if pcd.ScopeDecodeOffset < 0 {
			return nil, fmt.Errorf("unexpected negative scope decode offset for PCDesc %d: %d", i, pcd.ScopeDecodeOffset)
		}
		if pcd.ScopeDecodeOffset == 0 {
			lastOffset = uint64(pcd.PCOffset)
			continue
		}
		nodeID, err := visit(uint32(pcd.ScopeDecodeOffset))
		if err != nil {
			return nil, fmt.Errorf("failed to process scope descriptor chain for PCDesc %d: %w", i, err)
		}
		end := uint64(pcd.PCOffset) + 1
		table.Instructions = append(table.Instructions, &jvmsupp.InstructionRange{
			Begin:    lastOffset,
			End:      end,
			TreeNode: uint32(nodeID),
		})
		lastOffset = end
	}

	table.StringTable = make([]string, methodCounter)

	for methodName, methodID := range knownMethods {
		if methodName > uint32(len(nmeta.Metadata)) {
			return nil, fmt.Errorf("method index out of metadata range: %d", methodName)
		}
		if methodName <= 0 {
			return nil, fmt.Errorf("non-positive method index: %d", methodName)
		}

		methodAddr := nmeta.Metadata[methodName-1]

		methodObj := jvmbindings.Method(jvm.MakeObjPointer(uintptr(methodAddr)))

		nameBytes, err := readMethodName(methodObj, methodNameLimit)
		if err != nil {
			return nil, fmt.Errorf("failed to read method name for method %d (%x): %w", methodName, methodAddr, err)
		}
		table.StringTable[methodID] = string(nameBytes)
	}
	return table, nil
}
