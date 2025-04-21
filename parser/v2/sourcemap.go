package parser

import (
    "fmt"
    "strings"
    "unicode/utf8"

    "github.com/a-h/parse"
)

// NewSourceMap creates a new lookup to map templ source code to items in the parsed template.
func NewSourceMap() *SourceMap {
    return &SourceMap{
        SourceLinesToTarget:       make(map[uint32]map[uint32]parse.Position),
        TargetLinesToSource:       make(map[uint32]map[uint32]parse.Position),
        SourceSymbolRangeToTarget: make(map[uint32]map[uint32]Range),
        TargetSymbolRangeToSource: make(map[uint32]map[uint32]Range),
    }
}

type SourceMap struct {
    Expressions               []string
    SourceLinesToTarget       map[uint32]map[uint32]parse.Position
    TargetLinesToSource       map[uint32]map[uint32]parse.Position
    SourceSymbolRangeToTarget map[uint32]map[uint32]Range
    TargetSymbolRangeToSource map[uint32]map[uint32]Range
}

// AddSymbolRange adds a symbol range mapping between source (templ) and target (Go).
func (sm *SourceMap) AddSymbolRange(src, tgt Range) error {
    if src.From.Line == 0 || src.From.Col == 0 || src.To.Line == 0 || src.To.Col == 0 {
        return fmt.Errorf("invalid source range: %+v", src)
    }
    if tgt.From.Line == 0 || tgt.From.Col == 0 || tgt.To.Line == 0 || tgt.To.Col == 0 {
        return fmt.Errorf("invalid target range: %+v", tgt)
    }

    srcLine := uint32(src.From.Line)
    srcCol := uint32(src.From.Col)
    tgtLine := uint32(tgt.From.Line)
    tgtCol := uint32(tgt.From.Col)

    if _, ok := sm.SourceSymbolRangeToTarget[srcLine]; !ok {
        sm.SourceSymbolRangeToTarget[srcLine] = make(map[uint32]Range)
    }
    sm.SourceSymbolRangeToTarget[srcLine][srcCol] = tgt

    if _, ok := sm.TargetSymbolRangeToSource[tgtLine]; !ok {
        sm.TargetSymbolRangeToSource[tgtLine] = make(map[uint32]Range)
    }
    sm.TargetSymbolRangeToSource[tgtLine][tgtCol] = src

    return nil
}

// SymbolTargetRangeFromSource looks up the target (Go) range from a source (templ) position.
func (sm *SourceMap) SymbolTargetRangeFromSource(line, col uint32) (tgt Range, ok bool) {
    if line == 0 || col == 0 {
        return Range{}, false
    }
    lm, ok := sm.SourceSymbolRangeToTarget[line]
    if !ok {
        return
    }
    // Try exact match
    tgt, ok = lm[col]
    if ok {
        return
    }
    // Search nearby columns (±10) for closest match
    for i := uint32(1); i <= 10; i++ {
        if tgt, ok = lm[col+i]; ok {
            return
        }
        if col >= i {
            if tgt, ok = lm[col-i]; ok {
                return
            }
        }
    }
    return
}

// SymbolSourceRangeFromTarget looks up the source (templ) range from a target (Go) position.
func (sm *SourceMap) SymbolSourceRangeFromTarget(line, col uint32) (src Range, ok bool) {
    if line == 0 || col == 0 {
        return Range{}, false
    }
    lm, ok := sm.TargetSymbolRangeToSource[line]
    if !ok {
        return
    }
    // Try exact match
    if src, ok = lm[col]; ok {
        return
    }
    // Search nearby columns (±10) for closest match
    for i := uint32(1); i <= 10; i++ {
        if src, ok = lm[col+i]; ok {
            return
        }
        if col >= i {
            if src, ok = lm[col-i]; ok {
                return
            }
        }
    }
    // If no match, try previous lines (up to 5 lines back)
    for j := uint32(1); j <= 5 && line >= j; j++ {
        lm, ok = sm.TargetSymbolRangeToSource[line-j]
        if !ok {
            continue
        }
        if src, ok = lm[col]; ok {
            return
        }
        for i := uint32(1); i <= 10; i++ {
            if src, ok = lm[col+i]; ok {
                return
            }
            if col >= i {
                if src, ok = lm[col-i]; ok {
                    return
                }
            }
        }
    }
    return
}

// Add an item to the lookup.
func (sm *SourceMap) Add(src Expression, tgt Range) (updatedFrom parse.Position, err error) {
    if src.Value == "" {
        return parse.Position{}, fmt.Errorf("empty expression value")
    }
    if src.Range.From.Line == 0 || src.Range.From.Col == 0 || tgt.From.Line == 0 || tgt.From.Col == 0 {
        return parse.Position{}, fmt.Errorf("invalid source or target position: src=%+v, tgt=%+v", src.Range, tgt)
    }

    sm.Expressions = append(sm.Expressions, src.Value)
    srcIndex := int64(src.Range.From.Index)
    tgtIndex := int64(tgt.From.Index)

    lines := strings.Split(src.Value, "\n")
    for lineIndex, line := range lines {
        srcLine := uint32(src.Range.From.Line) + uint32(lineIndex)
        tgtLine := uint32(tgt.From.Line) + uint32(lineIndex)

        var srcCol, tgtCol uint32
        if lineIndex == 0 {
            // First line can have an offset.
            srcCol += uint32(src.Range.From.Col)
            tgtCol += uint32(tgt.From.Col)
        }

        // Process the cols.
        for _, r := range line {
            if _, ok := sm.SourceLinesToTarget[srcLine]; !ok {
                sm.SourceLinesToTarget[srcLine] = make(map[uint32]parse.Position)
            }
            sm.SourceLinesToTarget[srcLine][srcCol] = parse.Position{
                Index: int(tgtIndex),
                Line:  int(tgtLine),
                Col:   int(tgtCol),
            }

            if _, ok := sm.TargetLinesToSource[tgtLine]; !ok {
                sm.TargetLinesToSource[tgtLine] = make(map[uint32]parse.Position)
            }
            sm.TargetLinesToSource[tgtLine][tgtCol] = parse.Position{
                Index: int(srcIndex),
                Line:  int(srcLine),
                Col:   int(srcCol),
            }

            // Ignore invalid runes.
            rlen := utf8.RuneLen(r)
            if rlen < 0 {
                rlen = 1
            }
            srcCol += uint32(rlen)
            tgtCol += uint32(rlen)
            srcIndex += int64(rlen)
            tgtIndex += int64(rlen)
        }

        // LSPs include the newline char as a col.
        if _, ok := sm.SourceLinesToTarget[srcLine]; !ok {
            sm.SourceLinesToTarget[srcLine] = make(map[uint32]parse.Position)
        }
        sm.SourceLinesToTarget[srcLine][srcCol] = parse.Position{
            Index: int(tgtIndex),
            Line:  int(tgtLine),
            Col:   int(tgtCol),
        }

        if _, ok := sm.TargetLinesToSource[tgtLine]; !ok {
            sm.TargetLinesToSource[tgtLine] = make(map[uint32]parse.Position)
        }
        sm.TargetLinesToSource[tgtLine][tgtCol] = parse.Position{
            Index: int(srcIndex),
            Line:  int(srcLine),
            Col:   int(srcCol),
        }

        srcIndex++
        tgtIndex++
    }
    return src.Range.From, nil
}

// TargetPositionFromSource looks up the target position using the source position.
func (sm *SourceMap) TargetPositionFromSource(line, col uint32) (tgt parse.Position, ok bool) {
    if line == 0 || col == 0 {
        return parse.Position{}, false
    }
    lm, ok := sm.SourceLinesToTarget[line]
    if !ok {
        return
    }
    tgt, ok = lm[col]
    return
}

// SourcePositionFromTarget looks up the source position using the target position.
// If a source exists on the line but not the col, the function will search backwards.
func (sm *SourceMap) SourcePositionFromTarget(line, col uint32) (src parse.Position, ok bool) {
    if line == 0 || col == 0 {
        return parse.Position{}, false
    }
    lm, ok := sm.TargetLinesToSource[line]
    if !ok {
        return
    }
    for {
        src, ok = lm[col]
        if ok || col == 0 {
            return
        }
        col--
    }
}