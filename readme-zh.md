# Meow++ (MPP2)

Meow++ 是一個以 Go 實作的小型直譯式程式語言。此儲存庫包含 MPP2 執行階段、語法解析器、REPL、AST 匯出功能，以及將 Meow++ 原始碼包進產生出的 Go 程式中的原生編譯流程。

目前狀態：Version 2.0.0 Alpha  
主要原始碼副檔名：`.mpp`  
AST 副檔名：`.mst`（JSON）  
執行模型：以 magic method 為核心的動態物件運算與派發

## 概要

此儲存庫中的實作主要由三個層次構成：

- 將 `.mpp` 原始碼轉換成 JSON 風格命令樹的語法解析器。
- 透過型別物件與 magic method 來求值命令與值的執行階段。
- 可執行原始碼、執行預先產生 AST、匯出 AST、啟動 REPL，以及編譯為原生執行檔的 CLI。

## 儲存庫結構

| 路徑 | 用途 |
| --- | --- |
| `main.go` | CLI 入口點與命令分派。 |
| `parser/` | 敘述切分、正規表示式定義，以及 AST 產生。 |
| `runtime/` | 物件系統、求值器、內建功能、運算子與控制流程執行。 |
| `utils/` | 執行原始碼、執行 AST、REPL、AST 匯出與 compile 的高階輔助函式。 |

## CLI

執行檔接受子命令或直接指定檔案路徑。目前實作提供以下命令：

```text
mpp2 help
mpp2 run <file.mpp>
mpp2 run <file.mst>
mpp2 ast <file.mpp> <file.mst>
mpp2 repl
mpp2 compile <file.mpp> <output_binary>
mpp2 version
```

你也可以不經過 `run` 子命令，直接執行檔案：

```text
mpp2 program.mpp
mpp2 program.mst
```

一般的發佈包預期會包含 Go 原始碼、預先編譯好的執行檔，以及這份 README。因此這份文件會說明執行介面與原始碼結構，但不假設額外附帶其他封裝腳本。

## 如何把執行檔加入 PATH

如果你想在任何終端機位置都能直接執行 `mpp2`，可以把執行檔放進已經在 `PATH` 裡的資料夾，或把執行檔所在的資料夾加入 `PATH`。

### macOS 與 Linux

常見做法是，如果你有權限，就把執行檔放進 `/usr/local/bin`：

```sh
sudo cp mpp2 /usr/local/bin/mpp2
sudo chmod +x /usr/local/bin/mpp2
```

如果你想把執行檔放在家目錄底下，可以建立個人 bin 資料夾並將它加入 shell 設定：

```sh
mkdir -p $HOME/.local/bin
cp mpp2 $HOME/.local/bin/mpp2
chmod +x $HOME/.local/bin/mpp2
```

接著把下面這行加入 `~/.zshrc` 或 `~/.bashrc`：

```sh
export PATH="$HOME/.local/bin:$PATH"
```

重新載入 shell 後，可用以下命令確認：

```sh
mpp2 help
```

### Windows

可將 `mpp2.exe` 放進固定資料夾，例如 `C:\Tools\mpp2`，再到系統設定中把該資料夾加入使用者 PATH。

1. 開啟 **System Properties**。
2. 進入 **Environment Variables**。
3. 選取使用者的 **Path** 變數。
4. 新增包含 `mpp2.exe` 的資料夾。
5. 開新終端機並執行 `mpp2 help`。

也可以用 PowerShell 設定：

```powershell
$target = "$HOME\AppData\Local\mpp2"
New-Item -ItemType Directory -Force -Path $target
Copy-Item .\mpp2.exe "$target\mpp2.exe" -Force
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$target", "User")
```

## 語言基礎

Meow++ 是一個以表達式為核心、動態型別的語言。解析器目前可辨識以下主要敘述形式：

- 指定：`name = value`
- 複合指定：`+=`、`-=`、`*=`、`/=`、`%=`、`^=`、`&&=`、`||=`、`^^=`
- 刪除：`delete target`
- 條件分支：`if`、`else if`、`else`
- 迴圈：C 風格 `for`、foreach 風格 `for`、`while`、`do ... while`
- 函式：具名 `func` 與匿名 `func`
- 流程控制：`return`、`break`、`continue`、label 與 `goto`
- 匯入：使用同一套 `import "..."` 語法匯入原始碼模組或內建模組

### 程式入口

`!Meow++` 建議放在第一行，但也可以不使用。

加入這個標記的主要原因是延續語言相容性。Meow++ 1 使用必須且唯一的 `!Meow` 作為入口點，而入口點前面的程式只用來放定義、不會執行。Meow++ 2 為了繼承原本的語言特性，才保留了這個非強制性的入口標記。

如果程式中存在 `!Meow++` 入口點，執行時會跳過第一個 `!Meow++` 之前的所有程式碼。被跳過的部分不會執行，其中的定義也不會被載入。

### 常值與值

- 數字：`1`、`3.14`、`0xFF`
- 字串：原始碼中使用一般雙引號字串
- 布林值：`true`、`false`
- 空值：`null`
- 陣列：`[1, 2, 3]`
- 對應表：`{"name": "Meow", "year": 2026}`
- 屬性存取：`obj.attr`
- 索引存取：`obj[key]`
- 函式呼叫：`fn(a, b)`

### 運算子

目前解析器與執行階段支援以下幾類運算子：

- 算術：`+`、`-`、`*`、`/`、`%`、`^`
- 比較：`<`、`>`、`<=`、`>=`、`==`、`!=`
- 邏輯：`&&`、`||`、`^^`、一元 `!`
- 一元數值：一元 `+` 與一元 `-`
- 三元運算：`cond ? a : b`
- 類 shift 的 magic operator：`<<` 與 `>>`
- 解析器也可辨識遞增與遞減表達式。

`<<` 與 `>>` 在此實作中特別重要，因為標準串流物件就是透過它們來完成輸出與輸入。

### 範例

```text
!Meow++
import "math"

target = math.random.int(1, 100)
stdout << "Guess a number between 1 and 100!\n"
guess = ""

while (true) {
    stdout << "> "
    stdin >> guess
    guess_num = number(guess)
    if (guess_num < target) {
        stdout << "  Too low!\n"
    } else if (guess_num > target) {
        stdout << "  Too high!\n"
    } else {
        stdout << "  Correct!\n"
        break
    }
}
```

## 內建型別

執行階段目前提供以下核心物件型別：

- `number`
- `string`
- `bool`
- `null`
- `array`
- `map`
- `function`
- `subruntime`

各種行為由每個型別的 magic method 實作，因此運算子、型別轉換、屬性存取、索引操作與呼叫都會透過動態物件模型派發。

## 內建變數與函式

| 名稱 | 說明 |
| --- | --- |
| `stdout` | 可寫入的串流物件，使用 `<<` 輸出。 |
| `stdin` | 可讀取的串流物件，使用 `>>` 將輸入讀入目標。 |
| `stderr` | 錯誤輸出串流，使用 `<<` 輸出到 stderr。 |
| `number(x)` | 將值轉成數字。 |
| `string(x)` | 將值轉成字串。 |
| `array(x)` | 將值轉成陣列。 |
| `map(x)` | 將值轉成對應表。 |
| `len(x)` | 回傳值的長度。 |
| `type(x)` | 回傳執行時型別名稱。 |
| `set_magic(obj, name, fn)` | 替物件附加自訂 magic method。 |
| `set_type(obj, name)` | 覆寫物件的執行時型別字串。 |
| `exit(code?)` | 以可選的數值代碼結束程序。 |

## 內建模組

內建模組使用與檔案模組相同的匯入語法，例如 `import "math"`。

有些內建模組會直接修改你傳入的物件。例如 `array.append` 與 `array.pop` 會更新原本的陣列物件。

| 模組 | 目前實作中可用的成員 |
| --- | --- |
| `builtins` | 將所有內建變數與函式作為模組命名空間匯出。 |
| `unicode` | `upper`、`lower`、`ords`、`chrs` |
| `fileio` | `write`、`read`、`exists` |
| `memory` | `raw`、`size`、`usage` |
| `format` | `sprintf`、`sscanf` |
| `stdio` | `print`、`println`、`input` |
| `math` | `random.random`、`random.uniform`、`random.int`、`random.gauss`、`sin`、`cos`、`tan`、`asin`、`acos`、`atan`、`pi`、`e`、`sqrt`、`log`、`ln`、`log10`、`log2`、`floor`、`ceil`、`round`、`trunc`、`abs`、`min`、`max`、`exp` |
| `makeclass` | `get_info`、`Magic`，用於執行時物件資訊檢視與自訂 magic 建構 |
| `array` | `append`、`pop`、`slice`，用於陣列修改與切片 |

## AST 工作流程

解析器可以把原始碼檔匯出成 `.mst` 檔。該檔案是純 JSON，並可直接由執行階段執行。

```sh
cd src
mpp2 ast ../test.mpp ./test.mst
mpp2 run ./test.mst
```

這對應到 `utils.ParseMppAst` 與 `utils.RunMppAst` 的目前實作。

## REPL

REPL 會保留持續存在的 runtime，在括號尚未配對完成時接受多行輸入，並在求值失敗時回復 runtime 狀態。使用方式：

```sh
cd src
mpp2 repl
```

## 原生編譯

`compile` 命令並不是直接把 Meow++ 編譯成機器碼。它會先產生一個暫時性的 Go 程式，將原始碼嵌入其中，並透過 Meow++ runtime 執行，再把那個 Go 程式編譯成原生執行檔。

**注意：**如果你是從 Go 原始碼樹進行操作，compile 流程會依賴 Meow++ runtime source 已存在於應用程式環境目錄之下。

```sh
cd src
mpp2 compile ../test.mpp ../test_binary
```

## 注意事項與限制

- 此儲存庫目前仍標示為 Alpha。
- 此文件描述的是目前 Go 原始碼中的實作行為，而不是外部正式語言規格。
- 內建行為與使用者自訂行為都大量依賴 runtime magic method。
- 一般發佈包通常只需要執行檔、Go 原始碼樹與這份 README。
- 如果你要重新建置或從原始碼使用 `compile`，Meow++ 環境目錄中仍然需要存在 runtime source tree。

[GitHub 儲存庫](https://github.com/shp0717/mpp2)  
[English README](./readme.md)
