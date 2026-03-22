package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"byteflow-studio/internal/workflow"
)

func makeBlock(id, blockType string, cat BlockCategory) workflow.BlockDef {
	return workflow.BlockDef{ID: id, Type: blockType, Category: string(cat)}
}

func makeConn(id, fromBlock, fromPort, toBlock, toPort string) workflow.ConnectionDef {
	return workflow.ConnectionDef{
		ID:          id,
		FromBlockID: fromBlock,
		FromPortID:  fromPort,
		ToBlockID:   toBlock,
		ToPortID:    toPort,
	}
}

func TestValidLinearGraph(t *testing.T) {
	blocks := []workflow.BlockDef{
		makeBlock("input1", "uart", CategoryInput),
		makeBlock("analysis1", "value-display", CategoryAnalysis),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "input1", "out", "analysis1", "in-raw"),
	}
	portTypes := map[string]map[string]DataType{
		"input1":    {"out": DataTypeRaw},
		"analysis1": {"in-raw": DataTypeRaw},
	}

	order, err := ValidateGraph(blocks, conns, portTypes)
	require.NoError(t, err)
	assert.Equal(t, []string{"input1", "analysis1"}, order)
}

func TestDirectCycleDetected(t *testing.T) {
	blocks := []workflow.BlockDef{
		makeBlock("a", "proc", CategoryProcessing),
		makeBlock("b", "proc", CategoryProcessing),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "a", "out", "b", "in"),
		makeConn("c2", "b", "out", "a", "in"),
	}
	portTypes := map[string]map[string]DataType{
		"a": {"out": DataTypeNum, "in": DataTypeNum},
		"b": {"out": DataTypeNum, "in": DataTypeNum},
	}

	_, err := ValidateGraph(blocks, conns, portTypes)
	assert.ErrorContains(t, err, "cycle detected")
}

func TestIndirectCycleDetected(t *testing.T) {
	blocks := []workflow.BlockDef{
		makeBlock("a", "proc", CategoryProcessing),
		makeBlock("b", "proc", CategoryProcessing),
		makeBlock("c", "proc", CategoryProcessing),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "a", "out", "b", "in"),
		makeConn("c2", "b", "out", "c", "in"),
		makeConn("c3", "c", "out", "a", "in"),
	}
	portTypes := map[string]map[string]DataType{
		"a": {"out": DataTypeNum, "in": DataTypeNum},
		"b": {"out": DataTypeNum, "in": DataTypeNum},
		"c": {"out": DataTypeNum, "in": DataTypeNum},
	}

	_, err := ValidateGraph(blocks, conns, portTypes)
	assert.ErrorContains(t, err, "cycle detected")
}

func TestIncompatiblePortTypes(t *testing.T) {
	blocks := []workflow.BlockDef{
		makeBlock("input1", "uart", CategoryInput),
		makeBlock("proc1", "moving-average", CategoryProcessing),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "input1", "out", "proc1", "in"),
	}
	portTypes := map[string]map[string]DataType{
		"input1": {"out": DataTypeRaw},  // raw-bytes
		"proc1":  {"in": DataTypeNum},   // numeric — incompatible
	}

	_, err := ValidateGraph(blocks, conns, portTypes)
	assert.ErrorContains(t, err, "incompatible port types")
}

func TestNoValidInputToAnalysisPath(t *testing.T) {
	// Processing block with no connection to any analysis
	blocks := []workflow.BlockDef{
		makeBlock("input1", "uart", CategoryInput),
		makeBlock("proc1", "moving-average", CategoryProcessing),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "input1", "out", "proc1", "in"),
	}
	portTypes := map[string]map[string]DataType{
		"input1": {"out": DataTypeRaw},
		"proc1":  {"in": DataTypeRaw, "out": DataTypeNum},
	}

	_, err := ValidateGraph(blocks, conns, portTypes)
	assert.ErrorContains(t, err, "no valid input-to-analysis path")
}

func TestFanOutGraph(t *testing.T) {
	blocks := []workflow.BlockDef{
		makeBlock("input1", "simulator", CategoryInput),
		makeBlock("analysis1", "line-chart", CategoryAnalysis),
		makeBlock("analysis2", "value-display", CategoryAnalysis),
	}
	conns := []workflow.ConnectionDef{
		makeConn("c1", "input1", "out", "analysis1", "in"),
		makeConn("c2", "input1", "out", "analysis2", "in"),
	}
	portTypes := map[string]map[string]DataType{
		"input1":    {"out": DataTypeNum},
		"analysis1": {"in": DataTypeNum},
		"analysis2": {"in": DataTypeNum},
	}

	order, err := ValidateGraph(blocks, conns, portTypes)
	require.NoError(t, err)
	assert.Equal(t, "input1", order[0])
	assert.Len(t, order, 3)
}
