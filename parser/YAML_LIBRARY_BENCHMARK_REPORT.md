# YAML Library Benchmark Report

## `go.yaml.in/yaml/v4` vs `github.com/goccy/go-yaml`

**Date**: 2026-02-11
**Platform**: Linux (amd64), Intel Xeon Platinum 8581C @ 2.10GHz, 16 cores
**Go Version**: go1.24.7 linux/amd64
**Libraries Compared**:
| Library | Version | Import Path |
|---------|---------|-------------|
| yaml/v4 (current) | v4.0.0-rc.4 | `go.yaml.in/yaml/v4` |
| goccy/go-yaml | v1.19.2 | `github.com/goccy/go-yaml` |

**Methodology**: `go test -bench=BenchmarkYAMLCompare -benchmem -benchtime=2s -count=5`

---

## Test Fixtures (OAS 3.0.3 documents)

| Fixture | Size | Lines | Description |
|---------|------|-------|-------------|
| Small   | 1.3 KB | 58 | Minimal API: 2 paths, 3 operations |
| Medium  | 12.8 KB | 573 | Moderate API: users, posts, comments with schemas |
| Large   | 152.2 KB | 6,176 | Complex API: 50+ resources, deep schema nesting |

---

## Executive Summary

**`go.yaml.in/yaml/v4` is the clear winner across all benchmark categories**, typically running **1.5–2.8× faster** with **2.6–3.8× less memory** and **2.8–9.2× fewer allocations** than `goccy/go-yaml`.

The only area where goccy shows an advantage is **Marshal total bytes** — goccy allocates ~32% less memory for marshaling, but at the cost of **~9× more individual allocations**, which increases GC pressure.

### Recommendation

✅ **Keep `go.yaml.in/yaml/v4`** as the YAML library for oastools. Switching to `goccy/go-yaml` would result in significant performance regression across all primary operations.

---

## Detailed Results

### 1. Unmarshal (`[]byte` → `map[string]any`)

*The primary operation in the oastools parser — called for every YAML document parsed.*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **116,771** | **54,001** | **814** |
| Small | goccy | 230,432 | 142,956 | 2,252 |
| **Medium** | yamlv4 | **1,052,626** | **422,783** | **7,352** |
| Medium | goccy | 2,585,845 | 1,354,178 | 21,739 |
| **Large** | yamlv4 | **12,548,767** | **4,580,455** | **81,073** |
| Large | goccy | 25,181,605 | 15,958,752 | 244,736 |

**Speed ratio (goccy/yamlv4)**: Small 1.97×, Medium 2.46×, Large 2.01×
**Memory ratio**: Small 2.65×, Medium 3.20×, Large 3.48×
**Alloc ratio**: Small 2.77×, Medium 2.96×, Large 3.02×

---

### 2. Marshal (`map[string]any` → `[]byte`)

*Used in reference resolution when re-serializing documents after ref expansion.*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **191,804** | 169,415 | **374** |
| Small | goccy | 227,708 | **115,381** | 3,003 |
| **Medium** | yamlv4 | 2,530,922 | 1,614,036 | **3,285** |
| Medium | goccy | **2,404,686** | **1,162,902** | 29,634 |
| **Large** | yamlv4 | **14,737,603** | 20,192,746 | **36,139** |
| Large | goccy | 27,073,262 | **13,635,547** | 332,912 |

**Notable**: goccy uses 28–32% less total memory for marshaling but makes 8–9× more allocations. At medium size, goccy is marginally faster (5%). At small and large sizes, yamlv4 wins on speed.

**Speed ratio**: Small 1.19×, Medium 0.95× (goccy wins), Large 1.84×
**Memory ratio**: Small 0.68× (goccy wins), Medium 0.72× (goccy wins), Large 0.68× (goccy wins)
**Alloc ratio**: Small 8.03×, Medium 9.02×, Large 9.21×

---

### 3. Round-Trip (Unmarshal + Marshal)

*Reflects the full reference resolution cycle: parse → resolve → re-serialize.*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **336,830** | **223,430** | **1,188** |
| Small | goccy | 515,282 | 259,201 | 5,259 |
| **Medium** | yamlv4 | **3,589,327** | **2,036,861** | **10,638** |
| Medium | goccy | 5,572,377 | 2,537,464 | 51,388 |
| **Large** | yamlv4 | **29,027,536** | **24,773,255** | **117,213** |
| Large | goccy | 54,798,938 | 29,596,907 | 577,674 |

**Speed ratio**: Small 1.53×, Medium 1.55×, Large 1.89×
**Memory ratio**: Small 1.16×, Medium 1.25×, Large 1.19×
**Alloc ratio**: Small 4.43×, Medium 4.83×, Large 4.93×

---

### 4. Node/AST Parse (structural tree parsing)

*Used for source map building and order-preserving marshaling in oastools.*
*yamlv4: `yaml.Unmarshal` into `yaml.Node` / goccy: `parser.ParseBytes` into `ast.File`*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **96,340** | **40,237** | **538** |
| Small | goccy | 201,167 | 105,245 | 1,774 |
| **Medium** | yamlv4 | **835,583** | **299,064** | **4,725** |
| Medium | goccy | 2,344,391 | 1,030,673 | 17,537 |
| **Large** | yamlv4 | **9,364,323** | **3,266,596** | **52,345** |
| Large | goccy | 21,216,704 | 12,556,005 | 199,627 |

**Speed ratio**: Small 2.09×, Medium 2.81×, Large 2.27×
**Memory ratio**: Small 2.62×, Medium 3.45×, Large 3.84×
**Alloc ratio**: Small 3.30×, Medium 3.71×, Large 3.81×

---

### 5. Unmarshal to Struct (typed deserialization)

*Simulates deserializing into Go structs with yaml tags.*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **135,516** | **52,681** | **788** |
| Small | goccy | 281,450 | 142,638 | 2,249 |
| **Medium** | yamlv4 | **1,069,015** | **373,922** | **6,276** |
| Medium | goccy | 2,909,708 | 1,278,989 | 20,650 |
| **Large** | yamlv4 | **11,304,624** | **4,010,222** | **67,676** |
| Large | goccy | 24,981,041 | 15,058,876 | 231,255 |

**Speed ratio**: Small 2.08×, Medium 2.72×, Large 2.21×
**Memory ratio**: Small 2.71×, Medium 3.42×, Large 3.76×
**Alloc ratio**: Small 2.85×, Medium 3.29×, Large 3.42×

---

### 6. Streaming Decoder (`io.Reader`-based)

*Tests the decoder interface used by `ParseReader` in the parser package.*

| Size | Library | ns/op | B/op | allocs/op |
|------|---------|------:|-----:|----------:|
| **Small** | yamlv4 | **124,491** | **54,003** | **815** |
| Small | goccy | 251,832 | 143,067 | 2,253 |
| **Medium** | yamlv4 | **1,100,021** | **422,804** | **7,353** |
| Medium | goccy | 2,736,425 | 1,356,729 | 21,742 |
| **Large** | yamlv4 | **12,889,578** | **4,580,461** | **81,074** |
| Large | goccy | 26,066,233 | 15,958,895 | 244,736 |

**Speed ratio**: Small 2.02×, Medium 2.49×, Large 2.02×
**Memory ratio**: Small 2.65×, Medium 3.21×, Large 3.48×
**Alloc ratio**: Small 2.76×, Medium 2.96×, Large 3.02×

---

## Summary Heatmap

Ratio = `goccy / yamlv4` (higher = yamlv4 wins by more)

| Category | Speed (Small) | Speed (Med) | Speed (Large) | Memory (Large) | Allocs (Large) |
|----------|:-------------:|:-----------:|:-------------:|:--------------:|:--------------:|
| Unmarshal | 🟡 1.97× | 🔴 2.46× | 🔴 2.01× | 🔴 3.48× | 🔴 3.02× |
| Marshal | 🟢 1.19× | ✅ 0.95× | 🟡 1.84× | ✅ 0.68× | 🔴 9.21× |
| RoundTrip | 🟡 1.53× | 🟡 1.55× | 🟡 1.89× | 🟢 1.19× | 🔴 4.93× |
| NodeParse | 🔴 2.09× | 🔴 2.81× | 🔴 2.27× | 🔴 3.84× | 🔴 3.81× |
| UnmarshalStruct | 🔴 2.08× | 🔴 2.72× | 🔴 2.21× | 🔴 3.76× | 🔴 3.42× |
| Decoder | 🔴 2.02× | 🔴 2.49× | 🔴 2.02× | 🔴 3.48× | 🔴 3.02× |

**Legend**: ✅ goccy wins, 🟢 <1.5× (close), 🟡 1.5–2× (moderate), 🔴 >2× (significant)

---

## Key Observations

1. **yamlv4's C-based parser (libyaml) is fundamentally faster** — It uses a libyaml binding for lexing/parsing, which gives it a structural advantage over goccy's pure-Go lexer.

2. **goccy's allocation pattern is problematic** — While goccy sometimes uses less total bytes (especially in Marshal), it makes 3–9× more individual allocations. In a GC'd language like Go, allocation count directly impacts GC pause times and throughput.

3. **The performance gap widens with document size** — For medium-to-large documents (the real-world case for OAS specs), yamlv4's advantage is most pronounced: 2–2.8× faster on unmarshal, 3.5–3.8× less memory.

4. **Marshal is the only mixed result** — goccy's encoder allocates fewer total bytes but many more individual objects. For oastools' use case (where marshaling happens after ref resolution), the speed advantage of yamlv4 at large scale (1.84×) combined with 9× fewer allocations makes yamlv4 the better choice.

5. **Node parsing is critical for oastools** — Source maps and order preservation depend on AST/Node parsing. yamlv4 is 2.1–2.8× faster with 3.3–3.8× fewer allocations in this category.

---

## Raw Benchmark Output

```
goos: linux
goarch: amd64
pkg: github.com/erraggy/oastools/parser
cpu: INTEL(R) XEON(R) PLATINUM 8581C CPU @ 2.10GHz

BenchmarkYAMLCompare_Unmarshal/Small/yamlv4-16         	   21196	    115817 ns/op	   54002 B/op	     814 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/yamlv4-16         	   21780	    114435 ns/op	   54001 B/op	     814 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/yamlv4-16         	   21052	    116102 ns/op	   54001 B/op	     814 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/yamlv4-16         	   18992	    122940 ns/op	   54001 B/op	     814 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/yamlv4-16         	   21164	    114559 ns/op	   54002 B/op	     814 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/goccy-16          	   10000	    226853 ns/op	  142958 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/goccy-16          	   10000	    224505 ns/op	  142961 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/goccy-16          	   10000	    235822 ns/op	  142962 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/goccy-16          	   10000	    233496 ns/op	  142945 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Unmarshal/Small/goccy-16          	   10000	    231486 ns/op	  142956 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/yamlv4-16        	    2280	   1043997 ns/op	  422782 B/op	    7352 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/yamlv4-16        	    2210	   1033978 ns/op	  422782 B/op	    7352 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/yamlv4-16        	    2313	   1045203 ns/op	  422786 B/op	    7352 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/yamlv4-16        	    2396	   1077436 ns/op	  422782 B/op	    7352 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/yamlv4-16        	    2305	   1062517 ns/op	  422786 B/op	    7352 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/goccy-16         	     914	   2576885 ns/op	 1352050 B/op	   21737 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/goccy-16         	     920	   2566406 ns/op	 1354241 B/op	   21739 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/goccy-16         	     946	   2554972 ns/op	 1354375 B/op	   21739 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/goccy-16         	     952	   2647828 ns/op	 1355304 B/op	   21740 allocs/op
BenchmarkYAMLCompare_Unmarshal/Medium/goccy-16         	     981	   2583134 ns/op	 1355222 B/op	   21740 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/yamlv4-16         	     182	  12875200 ns/op	 4580446 B/op	   81073 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/yamlv4-16         	     199	  12069798 ns/op	 4580453 B/op	   81073 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/yamlv4-16         	     192	  12540969 ns/op	 4580471 B/op	   81073 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/yamlv4-16         	     195	  12562774 ns/op	 4580452 B/op	   81073 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/yamlv4-16         	     188	  12695092 ns/op	 4580451 B/op	   81073 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/goccy-16          	      94	  25386678 ns/op	15958712 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/goccy-16          	      93	  25569006 ns/op	15958758 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/goccy-16          	      90	  25364886 ns/op	15958805 B/op	  244735 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/goccy-16          	      88	  24708573 ns/op	15958659 B/op	  244735 allocs/op
BenchmarkYAMLCompare_Unmarshal/Large/goccy-16          	      94	  24878880 ns/op	15958828 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Marshal/Small/yamlv4-16           	   12968	    186674 ns/op	  169415 B/op	     374 allocs/op
BenchmarkYAMLCompare_Marshal/Small/yamlv4-16           	   12464	    194435 ns/op	  169416 B/op	     374 allocs/op
BenchmarkYAMLCompare_Marshal/Small/yamlv4-16           	   10000	    203231 ns/op	  169416 B/op	     374 allocs/op
BenchmarkYAMLCompare_Marshal/Small/yamlv4-16           	   12789	    188481 ns/op	  169414 B/op	     374 allocs/op
BenchmarkYAMLCompare_Marshal/Small/yamlv4-16           	   12783	    186200 ns/op	  169416 B/op	     374 allocs/op
BenchmarkYAMLCompare_Marshal/Small/goccy-16            	    9163	    225510 ns/op	  115391 B/op	    3003 allocs/op
BenchmarkYAMLCompare_Marshal/Small/goccy-16            	   10000	    236878 ns/op	  115377 B/op	    3003 allocs/op
BenchmarkYAMLCompare_Marshal/Small/goccy-16            	   10000	    226843 ns/op	  115397 B/op	    3004 allocs/op
BenchmarkYAMLCompare_Marshal/Small/goccy-16            	   10000	    222254 ns/op	  115370 B/op	    3003 allocs/op
BenchmarkYAMLCompare_Marshal/Small/goccy-16            	   10000	    227054 ns/op	  115372 B/op	    3003 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/yamlv4-16          	     946	   2559175 ns/op	 1614058 B/op	    3285 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/yamlv4-16          	     962	   2534100 ns/op	 1614037 B/op	    3285 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/yamlv4-16          	     996	   2501870 ns/op	 1614023 B/op	    3285 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/yamlv4-16          	     986	   2481855 ns/op	 1614022 B/op	    3285 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/yamlv4-16          	     961	   2577610 ns/op	 1614041 B/op	    3285 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/goccy-16           	     990	   2389735 ns/op	 1162647 B/op	   29633 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/goccy-16           	    1023	   2396853 ns/op	 1163016 B/op	   29636 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/goccy-16           	    1010	   2426404 ns/op	 1163336 B/op	   29634 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/goccy-16           	    1048	   2403521 ns/op	 1162901 B/op	   29633 allocs/op
BenchmarkYAMLCompare_Marshal/Medium/goccy-16           	    1017	   2406916 ns/op	 1162611 B/op	   29634 allocs/op
BenchmarkYAMLCompare_Marshal/Large/yamlv4-16           	     160	  14737744 ns/op	20192742 B/op	   36139 allocs/op
BenchmarkYAMLCompare_Marshal/Large/yamlv4-16           	     160	  14695931 ns/op	20192747 B/op	   36139 allocs/op
BenchmarkYAMLCompare_Marshal/Large/yamlv4-16           	     158	  15008052 ns/op	20192748 B/op	   36139 allocs/op
BenchmarkYAMLCompare_Marshal/Large/yamlv4-16           	     162	  14631888 ns/op	20192753 B/op	   36139 allocs/op
BenchmarkYAMLCompare_Marshal/Large/yamlv4-16           	     160	  14614399 ns/op	20192740 B/op	   36139 allocs/op
BenchmarkYAMLCompare_Marshal/Large/goccy-16            	      88	  27132490 ns/op	13635925 B/op	  332914 allocs/op
BenchmarkYAMLCompare_Marshal/Large/goccy-16            	      82	  27506283 ns/op	13635756 B/op	  332906 allocs/op
BenchmarkYAMLCompare_Marshal/Large/goccy-16            	      93	  26690662 ns/op	13634907 B/op	  332905 allocs/op
BenchmarkYAMLCompare_Marshal/Large/goccy-16            	      85	  27344631 ns/op	13636349 B/op	  332930 allocs/op
BenchmarkYAMLCompare_Marshal/Large/goccy-16            	      84	  26692242 ns/op	13634798 B/op	  332904 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/yamlv4-16         	    6090	    335551 ns/op	  223432 B/op	    1188 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/yamlv4-16         	    7364	    338631 ns/op	  223431 B/op	    1188 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/yamlv4-16         	    7130	    333456 ns/op	  223429 B/op	    1188 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/yamlv4-16         	    7444	    333187 ns/op	  223430 B/op	    1188 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/yamlv4-16         	    6188	    343326 ns/op	  223426 B/op	    1188 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/goccy-16          	    4644	    511674 ns/op	  259194 B/op	    5259 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/goccy-16          	    4650	    522585 ns/op	  259220 B/op	    5259 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/goccy-16          	    4678	    522515 ns/op	  259266 B/op	    5259 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/goccy-16          	    5011	    504660 ns/op	  259137 B/op	    5258 allocs/op
BenchmarkYAMLCompare_RoundTrip/Small/goccy-16          	    5049	    514976 ns/op	  259188 B/op	    5258 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/yamlv4-16        	     667	   3600523 ns/op	 2036874 B/op	   10638 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/yamlv4-16        	     644	   3592815 ns/op	 2036857 B/op	   10638 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/yamlv4-16        	     699	   3750660 ns/op	 2036840 B/op	   10638 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/yamlv4-16        	     652	   3517395 ns/op	 2036875 B/op	   10638 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/yamlv4-16        	     715	   3485242 ns/op	 2036857 B/op	   10638 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/goccy-16         	     441	   5477650 ns/op	 2536681 B/op	   51390 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/goccy-16         	     378	   5702724 ns/op	 2537651 B/op	   51386 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/goccy-16         	     440	   5667602 ns/op	 2537241 B/op	   51389 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/goccy-16         	     432	   5586720 ns/op	 2538108 B/op	   51388 allocs/op
BenchmarkYAMLCompare_RoundTrip/Medium/goccy-16         	     439	   5427191 ns/op	 2537641 B/op	   51388 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/yamlv4-16         	      85	  28186545 ns/op	24773217 B/op	  117212 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/yamlv4-16         	      72	  28727558 ns/op	24773264 B/op	  117213 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/yamlv4-16         	      75	  28663750 ns/op	24773250 B/op	  117213 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/yamlv4-16         	      70	  30587797 ns/op	24773285 B/op	  117214 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/yamlv4-16         	      74	  28972030 ns/op	24773259 B/op	  117213 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/goccy-16          	      48	  54534857 ns/op	29596402 B/op	  577685 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/goccy-16          	      44	  55109018 ns/op	29597511 B/op	  577673 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/goccy-16          	      44	  55056228 ns/op	29596873 B/op	  577670 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/goccy-16          	      44	  53573531 ns/op	29597009 B/op	  577682 allocs/op
BenchmarkYAMLCompare_RoundTrip/Large/goccy-16          	      50	  55721058 ns/op	29596742 B/op	  577661 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/yamlv4-16         	   25039	     95810 ns/op	   40238 B/op	     538 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/yamlv4-16         	   24361	     98040 ns/op	   40237 B/op	     538 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/yamlv4-16         	   25383	     94750 ns/op	   40236 B/op	     538 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/yamlv4-16         	   24412	     98020 ns/op	   40237 B/op	     538 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/yamlv4-16         	   24697	     95081 ns/op	   40237 B/op	     538 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/goccy-16          	   12272	    196591 ns/op	  105265 B/op	    1774 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/goccy-16          	   10000	    203097 ns/op	  105230 B/op	    1774 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/goccy-16          	   12068	    199129 ns/op	  105263 B/op	    1774 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/goccy-16          	   10000	    201145 ns/op	  105237 B/op	    1774 allocs/op
BenchmarkYAMLCompare_NodeParse/Small/goccy-16          	   10000	    205875 ns/op	  105232 B/op	    1774 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/yamlv4-16        	    2850	    844422 ns/op	  299062 B/op	    4725 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/yamlv4-16        	    2865	    854135 ns/op	  299064 B/op	    4725 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/yamlv4-16        	    2911	    804686 ns/op	  299064 B/op	    4725 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/yamlv4-16        	    2908	    832913 ns/op	  299064 B/op	    4725 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/yamlv4-16        	    3055	    841757 ns/op	  299065 B/op	    4725 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/goccy-16         	    1026	   2451661 ns/op	 1030782 B/op	   17537 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/goccy-16         	    1039	   2319676 ns/op	 1031418 B/op	   17538 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/goccy-16         	    1009	   2300366 ns/op	 1030251 B/op	   17537 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/goccy-16         	    1039	   2301357 ns/op	 1030802 B/op	   17537 allocs/op
BenchmarkYAMLCompare_NodeParse/Medium/goccy-16         	     964	   2348893 ns/op	 1030114 B/op	   17537 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/yamlv4-16         	     260	   9197537 ns/op	 3266572 B/op	   52345 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/yamlv4-16         	     256	   9381045 ns/op	 3266611 B/op	   52345 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/yamlv4-16         	     256	   9390222 ns/op	 3266603 B/op	   52345 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/yamlv4-16         	     242	   9511287 ns/op	 3266597 B/op	   52345 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/yamlv4-16         	     253	   9341522 ns/op	 3266598 B/op	   52345 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/goccy-16          	     100	  21102184 ns/op	12553620 B/op	  199627 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/goccy-16          	     100	  20989608 ns/op	12553809 B/op	  199628 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/goccy-16          	     100	  21047177 ns/op	12553680 B/op	  199627 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/goccy-16          	     100	  21428344 ns/op	12559329 B/op	  199627 allocs/op
BenchmarkYAMLCompare_NodeParse/Large/goccy-16          	     100	  21516205 ns/op	12559388 B/op	  199627 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/yamlv4-16   	   17500	    136278 ns/op	   52681 B/op	     788 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/yamlv4-16   	   17848	    137361 ns/op	   52682 B/op	     788 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/yamlv4-16   	   16504	    139658 ns/op	   52682 B/op	     788 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/yamlv4-16   	   18592	    130997 ns/op	   52681 B/op	     788 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/yamlv4-16   	   17920	    133288 ns/op	   52681 B/op	     788 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/goccy-16    	    9541	    281351 ns/op	  142608 B/op	    2249 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/goccy-16    	    8750	    285352 ns/op	  142629 B/op	    2249 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/goccy-16    	    8890	    276427 ns/op	  142660 B/op	    2249 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/goccy-16    	    9340	    282986 ns/op	  142652 B/op	    2249 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Small/goccy-16    	    8276	    281134 ns/op	  142639 B/op	    2249 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/yamlv4-16  	    2098	   1078192 ns/op	  373922 B/op	    6276 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/yamlv4-16  	    2217	   1061412 ns/op	  373924 B/op	    6276 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/yamlv4-16  	    2278	   1086440 ns/op	  373920 B/op	    6276 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/yamlv4-16  	    2216	   1060772 ns/op	  373921 B/op	    6276 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/yamlv4-16  	    2340	   1058261 ns/op	  373922 B/op	    6276 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/goccy-16   	     853	   2821647 ns/op	 1278501 B/op	   20649 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/goccy-16   	     836	   2903002 ns/op	 1279122 B/op	   20650 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/goccy-16   	     808	   2970334 ns/op	 1278760 B/op	   20650 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/goccy-16   	     822	   2934326 ns/op	 1279820 B/op	   20651 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Medium/goccy-16   	     793	   2919232 ns/op	 1278743 B/op	   20650 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/yamlv4-16   	     208	  11413397 ns/op	 4010240 B/op	   67676 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/yamlv4-16   	     212	  11281694 ns/op	 4010218 B/op	   67676 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/yamlv4-16   	     212	  11333325 ns/op	 4010219 B/op	   67676 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/yamlv4-16   	     217	  11193086 ns/op	 4010225 B/op	   67676 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/yamlv4-16   	     213	  11301619 ns/op	 4010207 B/op	   67676 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/goccy-16    	      87	  26231843 ns/op	15060360 B/op	  231255 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/goccy-16    	     100	  24828080 ns/op	15060089 B/op	  231255 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/goccy-16    	     100	  24829443 ns/op	15060152 B/op	  231255 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/goccy-16    	      90	  24518218 ns/op	15060199 B/op	  231255 allocs/op
BenchmarkYAMLCompare_UnmarshalStruct/Large/goccy-16    	      88	  24497621 ns/op	15053581 B/op	  231254 allocs/op
BenchmarkYAMLCompare_Decoder/Small/yamlv4-16           	   19454	    122242 ns/op	   54004 B/op	     815 allocs/op
BenchmarkYAMLCompare_Decoder/Small/yamlv4-16           	   19600	    124852 ns/op	   54004 B/op	     815 allocs/op
BenchmarkYAMLCompare_Decoder/Small/yamlv4-16           	   19509	    122236 ns/op	   54004 B/op	     815 allocs/op
BenchmarkYAMLCompare_Decoder/Small/yamlv4-16           	   19533	    127060 ns/op	   54003 B/op	     815 allocs/op
BenchmarkYAMLCompare_Decoder/Small/yamlv4-16           	   18810	    126064 ns/op	   54004 B/op	     815 allocs/op
BenchmarkYAMLCompare_Decoder/Small/goccy-16            	   10000	    256440 ns/op	  143085 B/op	    2253 allocs/op
BenchmarkYAMLCompare_Decoder/Small/goccy-16            	    9894	    252109 ns/op	  143079 B/op	    2253 allocs/op
BenchmarkYAMLCompare_Decoder/Small/goccy-16            	    9770	    251452 ns/op	  143090 B/op	    2253 allocs/op
BenchmarkYAMLCompare_Decoder/Small/goccy-16            	    9529	    248491 ns/op	  143059 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Decoder/Small/goccy-16            	   10000	    250666 ns/op	  143023 B/op	    2252 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/yamlv4-16          	    2226	   1067145 ns/op	  422799 B/op	    7353 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/yamlv4-16          	    2283	   1104113 ns/op	  422809 B/op	    7353 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/yamlv4-16          	    2163	   1100604 ns/op	  422803 B/op	    7353 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/yamlv4-16          	    2091	   1115938 ns/op	  422805 B/op	    7353 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/yamlv4-16          	    2155	   1112307 ns/op	  422806 B/op	    7353 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/goccy-16           	     746	   2841917 ns/op	 1356593 B/op	   21742 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/goccy-16           	     903	   2687153 ns/op	 1357093 B/op	   21742 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/goccy-16           	     874	   2725680 ns/op	 1356463 B/op	   21742 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/goccy-16           	     904	   2670439 ns/op	 1356401 B/op	   21742 allocs/op
BenchmarkYAMLCompare_Decoder/Medium/goccy-16           	     885	   2756935 ns/op	 1357097 B/op	   21742 allocs/op
BenchmarkYAMLCompare_Decoder/Large/yamlv4-16           	     184	  12977353 ns/op	 4580462 B/op	   81074 allocs/op
BenchmarkYAMLCompare_Decoder/Large/yamlv4-16           	     183	  12904234 ns/op	 4580454 B/op	   81074 allocs/op
BenchmarkYAMLCompare_Decoder/Large/yamlv4-16           	     186	  12850530 ns/op	 4580464 B/op	   81074 allocs/op
BenchmarkYAMLCompare_Decoder/Large/yamlv4-16           	     187	  12881012 ns/op	 4580466 B/op	   81074 allocs/op
BenchmarkYAMLCompare_Decoder/Large/yamlv4-16           	     184	  12834762 ns/op	 4580460 B/op	   81074 allocs/op
BenchmarkYAMLCompare_Decoder/Large/goccy-16            	      87	  26937756 ns/op	15958901 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Decoder/Large/goccy-16            	      94	  25815777 ns/op	15958898 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Decoder/Large/goccy-16            	      90	  25976016 ns/op	15958943 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Decoder/Large/goccy-16            	      86	  26202000 ns/op	15958762 B/op	  244736 allocs/op
BenchmarkYAMLCompare_Decoder/Large/goccy-16            	      97	  25399614 ns/op	15958972 B/op	  244736 allocs/op
PASS
ok  	github.com/erraggy/oastools/parser	431.010s
```
