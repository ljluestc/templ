package parser

import (
    "sort"
    "testing"

    "github.com/a-h/parse"
    "github.com/google/go-cmp/cmp"
)

// Test data.
//
//	|  - | 0 1 2 3 4 5 6 7 8 9
//	|  - | - - - - - - - - -
//	|  0 |
//	|  1 |   a b c d e f g h i
//	|  2 |   j k l m n o
//	|  3 |   p q r s t u v
//	|  4 |
//	|  5 |   w x y
//	|  6 |   z
//	|  7 |   m u l t i
//	|  8 | l i n e
//	|  9 | m a t c h
//	| 10 | 生 日 快 乐
func pos(index int64, line, col uint32) parse.Position {
    return parse.Position{
        Index: int(index),
        Line:  int(line),
        Col:   int(col),
    }
}

func rangePos(from, to parse.Position) Range {
    return Range{
        From: from,
        To:   to,
    }
}

func TestSourceMapPosition(t *testing.T) {
    var tests = []struct {
        name       string
        setup      func(sm *SourceMap)
        source     parse.Position
        target     parse.Position
        expectedOK bool
    }{
        {
            name: "searching within the map returns a result",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("abc", pos(0, 1, 1), pos(2, 1, 3)),
                    rangePos(pos(0, 5, 1), pos(2, 5, 3)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(0, 1, 1), // a
            target:     pos(0, 5, 1),
            expectedOK: true,
        },
        {
            name: "offsets within the match are handled",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("abc", pos(0, 1, 1), pos(2, 1, 3)),
                    rangePos(pos(0, 5, 1), pos(2, 5, 3)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(1, 1, 2), // b
            target:     pos(1, 5, 2),
            expectedOK: true,
        },
        {
            name: "the match that starts closest to the source is returned",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("rst", pos(4, 3, 3), pos(7, 3, 6)),
                    rangePos(pos(0, 8, 6), pos(3, 8, 9)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
                _, err = sm.Add(NewExpression("s", pos(5, 3, 4), pos(5, 3, 4)),
                    rangePos(pos(1, 8, 7), pos(1, 8, 7)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(5, 3, 4), // s
            target:     pos(1, 8, 7),
            expectedOK: true,
        },
        {
            name: "the start line within a multiline match is detected",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("multi\nline\nmatch", pos(0, 7, 0), pos(16, 9, 5)),
                    rangePos(pos(1, 1, 1), pos(17, 3, 5)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(0, 7, 0), // m (ulti)
            target:     pos(1, 1, 1),
            expectedOK: true,
        },
        {
            name: "the middle line within a multiline match is detected",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("multi\nline\nmatch", pos(0, 7, 0), pos(16, 9, 5)),
                    rangePos(pos(1, 1, 1), pos(17, 3, 5)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(6, 8, 1), // l (ine)
            target:     pos(7, 2, 1),
            expectedOK: true,
        },
        {
            name: "the final line within a multiline match is detected",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("multi\nline\nmatch", pos(0, 7, 0), pos(16, 9, 5)),
                    rangePos(pos(1, 1, 1), pos(17, 3, 5)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(12, 9, 1), // m (atch)
            target:     pos(13, 3, 1),
            expectedOK: true,
        },
        {
            name: "unicode characters are indexed correctly (sheng)",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("生日快乐", pos(0, 10, 1), pos(12, 10, 5)),
                    rangePos(pos(0, 11, 1), pos(12, 11, 5)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(0, 10, 1), // 生
            target:     pos(0, 11, 1),
            expectedOK: true,
        },
        {
            name: "unicode characters are indexed correctly (ri)",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("生日快乐", pos(0, 10, 1), pos(12, 10, 5)),
                    rangePos(pos(0, 11, 1), pos(12, 11, 5)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(3, 10, 2), // 日
            target:     pos(3, 11, 2),
            expectedOK: true,
        },
        {
            name: "out of bounds",
            setup: func(sm *SourceMap) {
                // No mappings added
            },
            source:     pos(0, 20, 10),
            target:     pos(0, 0, 0),
            expectedOK: false,
        },
        {
            name: "invalid position",
            setup: func(sm *SourceMap) {
                _, err := sm.Add(NewExpression("abc", pos(0, 1, 1), pos(2, 1, 3)),
                    rangePos(pos(0, 5, 1), pos(2, 5, 3)))
                if err != nil {
                    t.Fatalf("Add failed: %v", err)
                }
            },
            source:     pos(0, 0, 0),
            target:     pos(0, 0, 0),
            expectedOK: false,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            sm := NewSourceMap()
            tt.setup(sm)

            // Test source to target
            actualTarget, ok := sm.TargetPositionFromSource(uint32(tt.source.Line), uint32(tt.source.Col))
            if ok != tt.expectedOK {
                t.Errorf("TargetPositionFromSource: expected ok=%v, got %v", tt.expectedOK, ok)
            }
            if tt.expectedOK {
                diff := cmp.Diff(tt.target, actualTarget)
                if diff != "" {
                    dumpMappings(t, sm, "SourceLinesToTarget")
                    t.Errorf("TargetPositionFromSource\n%s", diff)
                }
            }

            // Test target to source
            if tt.expectedOK {
                actualSource, ok := sm.SourcePositionFromTarget(uint32(tt.target.Line), uint32(tt.target.Col))
                if !ok {
                    t.Fatalf("SourcePositionFromTarget: expected result, got no results")
                }
                diff := cmp.Diff(tt.source, actualSource)
                if diff != "" {
                    dumpMappings(t, sm, "TargetLinesToSource")
                    t.Errorf("SourcePositionFromTarget\n%s", diff)
                }
            }
        })
    }
}

func TestSourceMapSymbolRange(t *testing.T) {
    var tests = []struct {
        name       string
        setup      func(sm *SourceMap)
        source     Range
        target     Range
        sourceLine uint32
        sourceCol  uint32
        targetLine uint32
        targetCol  uint32
    }{
        {
            name: "component rename (hello world example)",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(10, 2, 6), pos(15, 2, 11)), // hello in hello.templ
                    rangePos(pos(200, 9, 5), pos(205, 9, 10)), // hello in hello_templ.go
                )
                if err != nil {
                    t.Fatalf("AddSymbolRange failed: %v", err)
                }
            },
            source:     rangePos(pos(10, 2, 6), pos(15, 2, 11)),
            target:     rangePos(pos(200, 9, 5), pos(205, 9, 10)),
            sourceLine: 2,
            sourceCol:  6,
            targetLine: 9,
            targetCol:  5,
        },
        {
            name: "single character symbol",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(0, 1, 1), pos(0, 1, 1)), // a in source
                    rangePos(pos(0, 5, 1), pos(0, 5, 1)), // a in target
                )
                if err != nil {
                    t.Fatalf("AddSymbolRange failed: %v", err)
                }
            },
            source:     rangePos(pos(0, 1, 1), pos(0, 1, 1)),
            target:     rangePos(pos(0, 5, 1), pos(0, 5, 1)),
            sourceLine: 1,
            sourceCol:  1,
            targetLine: 5,
            targetCol:  1,
        },
        {
            name: "multiline symbol",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(0, 7, 1), pos(16, 9, 5)), // multi\nline\nmatch
                    rangePos(pos(1, 1, 1), pos(17, 3, 5)),
                )
                if err != nil {
                    t.Fatalf("AddSymbolRange failed: %v", err)
                }
            },
            source:     rangePos(pos(0, 7, 1), pos(16, 9, 5)),
            target:     rangePos(pos(1, 1, 1), pos(17, 3, 5)),
            sourceLine: 7,
            sourceCol:  1,
            targetLine: 1,
            targetCol:  1,
        },
        {
            name: "nearby column lookup (gopls offset)",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(10, 2, 6), pos(15, 2, 11)), // hello in hello.templ
                    rangePos(pos(200, 9, 5), pos(205, 9, 10)), // hello in hello_templ.go
                )
                if err != nil {
                    t.Fatalf("AddSymbolRange failed: %v", err)
                }
            },
            source:     rangePos(pos(10, 2, 6), pos(15, 2, 11)),
            target:     rangePos(pos(200, 9, 5), pos(205, 9, 10)),
            sourceLine: 2,
            sourceCol:  6,
            targetLine: 9,
            targetCol:  7, // Offset by +2
        },
        {
            name: "nearby line lookup (gopls line shift)",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(10, 2, 6), pos(15, 2, 11)), // hello in hello.templ
                    rangePos(pos(200, 9, 5), pos(205, 9, 10)), // hello in hello_templ.go
                )
                if err != nil {
                    t.Fatalf("AddSymbolRange failed: %v", err)
                }
            },
            source:     rangePos(pos(10, 2, 6), pos(15, 2, 11)),
            target:     rangePos(pos(200, 9, 5), pos(205, 9, 10)),
            sourceLine: 2,
            sourceCol:  6,
            targetLine: 10, // Offset by +1 line
            targetCol:  5,
        },
        {
            name: "invalid range",
            setup: func(sm *SourceMap) {
                err := sm.AddSymbolRange(
                    rangePos(pos(0, 0, 0), pos(0, 0, 0)), // Invalid
                    rangePos(pos(0, 5, 1), pos(0, 5, 1)),
                )
                if err == nil {
                    t.Fatal("AddSymbolRange expected error for invalid range")
                }
            },
            source:     rangePos(pos(0, 0, 0), pos(0, 0, 0)),
            target:     rangePos(pos(0, 5, 1), pos(0, 5, 1)),
            sourceLine: 0,
            sourceCol:  0,
            targetLine: 0,
            targetCol:  0,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            sm := NewSourceMap()
            tt.setup(sm)

            // Test source to target
            actualTarget, ok := sm.SymbolTargetRangeFromSource(tt.sourceLine, tt.sourceCol)
            if ok != (tt.targetLine != 0 || tt.targetCol != 0) {
                t.Errorf("SymbolTargetRangeFromSource: expected ok=%v, got %v", tt.targetLine != 0 || tt.targetCol != 0, ok)
            }
            if ok {
                diff := cmp.Diff(tt.target, actualTarget)
                if diff != "" {
                    dumpSymbolMappings(t, sm, "SourceSymbolRangeToTarget")
                    t.Errorf("SymbolTargetRangeFromSource\n%s", diff)
                }
            }

            // Test target to source
            if tt.targetLine != 0 || tt.targetCol != 0 {
                actualSource, ok := sm.SymbolSourceRangeFromTarget(tt.targetLine, tt.targetCol)
                if !ok {
                    t.Errorf("SymbolSourceRangeFromTarget: expected result from target %v:%v, got no results", tt.targetLine, tt.targetCol)
                }
                diff := cmp.Diff(tt.source, actualSource)
                if diff != "" {
                    dumpSymbolMappings(t, sm, "TargetSymbolRangeToSource")
                    t.Errorf("SymbolSourceRangeFromTarget\n%s", diff)
                }
            }
        })
    }
}

// dumpMappings prints all position mappings for debugging.
func dumpMappings(t *testing.T, sm *SourceMap, mapName string) {
    t.Helper()
    if mapName == "SourceLinesToTarget" {
        lines := keys(sm.SourceLinesToTarget)
        sort.Slice(lines, func(i, j int) bool { return lines[i] < lines[j] })
        for _, line := range lines {
            cols := keys(sm.SourceLinesToTarget[line])
            sort.Slice(cols, func(i, j int) bool { return cols[i] < cols[j] })
            t.Logf("SourceLinesToTarget Line %d: Cols %v", line, cols)
            for _, col := range cols {
                t.Logf("  %d:%d -> %+v", line, col, sm.SourceLinesToTarget[line][col])
            }
        }
    } else if mapName == "TargetLinesToSource" {
        lines := keys(sm.TargetLinesToSource)
        sort.Slice(lines, func(i, j int) bool { return lines[i] < lines[j] })
        for _, line := range lines {
            cols := keys(sm.TargetLinesToSource[line])
            sort.Slice(cols, func(i, j int) bool { return cols[i] < cols[j] })
            t.Logf("TargetLinesToSource Line %d: Cols %v", line, cols)
            for _, col := range cols {
                t.Logf("  %d:%d -> %+v", line, col, sm.TargetLinesToSource[line][col])
            }
        }
    }
}

// dumpSymbolMappings prints all symbol range mappings for debugging.
func dumpSymbolMappings(t *testing.T, sm *SourceMap, mapName string) {
    t.Helper()
    if mapName == "SourceSymbolRangeToTarget" {
        lines := keys(sm.SourceSymbolRangeToTarget)
        sort.Slice(lines, func(i, j int) bool { return lines[i] < lines[j] })
        for _, line := range lines {
            cols := keys(sm.SourceSymbolRangeToTarget[line])
            sort.Slice(cols, func(i, j int) bool { return cols[i] < cols[j] })
            t.Logf("SourceSymbolRangeToTarget Line %d: Cols %v", line, cols)
            for _, col := range cols {
                t.Logf("  %d:%d -> %+v", line, col, sm.SourceSymbolRangeToTarget[line][col])
            }
        }
    } else if mapName == "TargetSymbolRangeToSource" {
        lines := keys(sm.TargetSymbolRangeToSource)
        sort.Slice(lines, func(i, j int) bool { return lines[i] < lines[j] })
        for _, line := range lines {
            cols := keys(sm.TargetSymbolRangeToSource[line])
            sort.Slice(cols, func(i, j int) bool { return cols[i] < cols[j] })
            t.Logf("TargetSymbolRangeToSource Line %d: Cols %v", line, cols)
            for _, col := range cols {
                t.Logf("  %d:%d -> %+v", line, col, sm.TargetSymbolRangeToSource[line][col])
            }
        }
    }
}

func keys[K comparable, V any](m map[K]V) (keys []K) {
    for k := range m {
        keys = append(keys, k)
    }
    return
}