#!/usr/bin/env python3
"""Migrate bootstrap .fg files from free-function Dict API to method syntax.

Strategy: line-by-line regex with careful matching.
"""
import re
import sys
import os
import glob

def find_matching_paren(s, start):
    """Find the closing paren matching the open paren at position start."""
    depth = 0
    i = start
    while i < len(s):
        if s[i] == '(':
            depth += 1
        elif s[i] == ')':
            depth -= 1
            if depth == 0:
                return i
        i += 1
    return -1

def split_top_level_args(s):
    """Split a string by commas, respecting nested parens/brackets/braces."""
    args = []
    depth = 0
    current = []
    for c in s:
        if c in '([{':
            depth += 1
            current.append(c)
        elif c in ')]}':
            depth -= 1
            current.append(c)
        elif c == ',' and depth == 0:
            args.append(''.join(current).strip())
            current = []
        else:
            current.append(c)
    if current:
        args.append(''.join(current).strip())
    return args

def migrate_dict_call(match, func_name):
    """Convert dict_XXX<Type>(dict, ...) to dict.XXX(...)"""
    full = match.group(0)
    # Find the opening paren after the type args
    paren_start = full.index('(')
    paren_end = find_matching_paren(full, paren_start)
    if paren_end < 0:
        return full  # can't parse, leave unchanged
    
    inner = full[paren_start+1:paren_end]
    args = split_top_level_args(inner)
    
    if len(args) < 1:
        return full
    
    dict_var = args[0]
    remaining = args[1:]
    
    method_name = func_name.replace('dict_', '')
    if remaining:
        return f"{dict_var}.{method_name}({', '.join(remaining)})"
    else:
        return f"{dict_var}.{method_name}()"

def migrate_line(line):
    """Migrate a single line."""
    original = line
    
    # dict_set<Type>(d, k, v) → d.set(k, v)
    # dict_get<Type>(d, k) → d.get(k)
    # dict_has<Type>(d, k) → d.has(k)
    # dict_remove<Type>(d, k) → d.remove(k)
    # dict_keys<Type>(d) → d.keys()
    for func in ['dict_set', 'dict_get', 'dict_has', 'dict_remove', 'dict_keys']:
        # With type args
        pattern = re.compile(rf'{func}<[^>]*>\(')
        while pattern.search(line):
            m = pattern.search(line)
            start = m.start()
            # Find matching closing paren
            paren_start = line.index('(', start + len(func))
            paren_end = find_matching_paren(line, paren_start)
            if paren_end < 0:
                break
            call_text = line[start:paren_end+1]
            inner = line[paren_start+1:paren_end]
            args = split_top_level_args(inner)
            if len(args) < 1:
                break
            dict_var = args[0]
            remaining = args[1:]
            method = func.replace('dict_', '')
            if remaining:
                replacement = f"{dict_var}.{method}({', '.join(remaining)})"
            else:
                replacement = f"{dict_var}.{method}()"
            line = line[:start] + replacement + line[paren_end+1:]
        
        # Without type args
        pattern2 = re.compile(rf'{func}\(')
        while pattern2.search(line):
            m = pattern2.search(line)
            start = m.start()
            paren_start = start + len(func)
            paren_end = find_matching_paren(line, paren_start)
            if paren_end < 0:
                break
            inner = line[paren_start+1:paren_end]
            args = split_top_level_args(inner)
            if len(args) < 1:
                break
            dict_var = args[0]
            remaining = args[1:]
            method = func.replace('dict_', '')
            if remaining:
                replacement = f"{dict_var}.{method}({', '.join(remaining)})"
            else:
                replacement = f"{dict_var}.{method}()"
            line = line[:start] + replacement + line[paren_end+1:]
    
    # dict_new<V>() → Dict<Sym, V>()
    line = re.sub(r'dict_new<([^>]+)>\(\)', r'Dict<Sym, \1>()', line)
    
    # Dict<V> → Dict<Sym, V>  (only single type param, not already Dict<X, Y>)
    line = re.sub(r'Dict<([^,>]+)>', r'Dict<Sym, \1>', line)
    # Fix double-migration
    line = re.sub(r'Dict<Sym, Sym,', 'Dict<Sym,', line)
    
    # sym("name") → `name`
    line = re.sub(r'sym\("([^"]+)"\)', r'`\1`', line)
    
    # sym_to_string(x) → x.get_name()  — sym_to_string was the old API
    # Actually check if this exists in bootstrap
    
    return line

def migrate_file(filepath):
    with open(filepath, 'r') as f:
        lines = f.readlines()
    
    new_lines = [migrate_line(line) for line in lines]
    
    if new_lines != lines:
        with open(filepath, 'w') as f:
            f.writelines(new_lines)
        return True
    return False

def main():
    bootstrap_dir = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'bootstrap')
    files = glob.glob(os.path.join(bootstrap_dir, '**/*.fg'), recursive=True)
    
    changed = 0
    for f in sorted(files):
        if migrate_file(f):
            print(f'  migrated: {os.path.relpath(f)}')
            changed += 1
    
    # Also check remaining occurrences
    remaining = 0
    for f in sorted(files):
        with open(f) as fh:
            for i, line in enumerate(fh, 1):
                for pat in ['dict_set', 'dict_get', 'dict_has', 'dict_remove', 'dict_keys', 'dict_new']:
                    if pat in line:
                        print(f'  REMAINING: {os.path.relpath(f)}:{i}: {line.rstrip()}')
                        remaining += 1
    
    print(f'\n{changed} files migrated, {remaining} remaining occurrences')

if __name__ == '__main__':
    main()
