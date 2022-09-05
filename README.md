Install with `go install` from within project directory.

Run with `anime-freq-gen` followed by the flags below.

| Flags | Required | Example | Description |
| ----- | -------- | ------- | ----------- |
| -in   | Yes      | -in /path/to/dir/or/subtitle-file | **Required** |
| -out  | No       | -out /path/to/directory | Output file *'freq.txt'* will be saved to *-out*. Defaults to directory of *-in*. *Optional* |
| -r    | No       | -r=false | Search recursively. Defaults to **true**. *Optional* |
| -v    | No       | -v=true | Verbosity. Defaults to **false**. *Optional* |
| -wl   | No       | -wl | Outputs a list of unique words without their frequencies. |

Doesn't currently allow a custom output filename. Always saves as frequencies.txt or words.txt.  
Output may not look nice if not viewed with a monospaced font.  

DISCLAIMER: This was made on a whim. There is no quality guarantee. There is no quality.  
