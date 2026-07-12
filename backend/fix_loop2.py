import subprocess, re, sys

BIG_FILES = {'internal/service/admin_service.go', 'internal/service/gateway_service.go', 'internal/service/openai_gateway_service.go', 'internal/service/setting_service.go'}

def extract_decl(line, code):
    idx = line - 1
    lines = code.split('\n')
    decl_start = idx
    while decl_start > 0:
        stripped = lines[decl_start].lstrip()
        if stripped.startswith('type ') or stripped.startswith('func ') or stripped.startswith('var ') or stripped.startswith('const '):
            break
        decl_start -= 1
    end = decl_start
    brace_count = 0
    in_body = False
    while end < len(lines):
        l = lines[end]
        if '{' in l:
            brace_count += l.count('{')
            in_body = True
        if '}' in l:
            brace_count -= l.count('}')
        if in_body and brace_count == 0:
            break
        end += 1
    return decl_start, end

def fix():
    changed = True
    iter = 0
    while changed and iter < 200:
        iter += 1
        print(f"=== Iteration {iter} ===")
        res = subprocess.run(['go', 'build', './...'], capture_output=True, text=True)
        if res.returncode == 0:
            print("Build succeeded!")
            return 0
        errs = res.stderr.strip().split('\n')
        removals = []
        i = 0
        while i < len(errs):
            err = errs[i]
            m = re.match(r'^\s*(\S+):(\d+):\d+:\s+(\S+)\s+redeclared in this block', err)
            if m and i + 1 < len(errs):
                file1, line1, name = m.group(1), int(m.group(2)), m.group(3)
                other = errs[i+1]
                m2 = re.match(r'^\s*(\S+):(\d+):\d+:\s+other declaration of \S+', other)
                if m2:
                    file2, line2 = m2.group(1), int(m2.group(2))
                    if file1 in BIG_FILES:
                        removals.append((file1, line1, name))
                    elif file2 in BIG_FILES:
                        removals.append((file2, line2, name))
                i += 2
            else:
                i += 1
        if not removals:
            print("No parseable redeclared errors. Remaining output:")
            print(res.stderr[:3000])
            return 1
        seen = set()
        unique_removals = []
        for r in removals:
            key = (r[0], r[1])
            if key not in seen:
                seen.add(key)
                unique_removals.append(r)
        unique_removals.sort(key=lambda x: (x[0], x[1]), reverse=True)
        for file, line, name in unique_removals:
            print(f"Removing {name} from {file}:{line}")
            with open(file, 'r') as f:
                code = f.read()
            start, end = extract_decl(line, code)
            print(f"  -> lines {start+1}-{end+1}")
            lines = code.split('\n')
            del lines[start:end+1]
            with open(file, 'w') as f:
                f.write('\n'.join(lines))
        changed = True
    print("Max iterations reached")
    return 1

sys.exit(fix())
