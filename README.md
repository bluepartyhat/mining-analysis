# Mining Analysis Tool for BitClout
mining-analysis.go returns a list of public keys who've mined blocks within a certain blockheight range.
The output also states how many blocks those public keys were responsible for mining.
The results can also be optionally exported to a csv file for analysis elsewhere.

## Setup
 mining-analysis requires bitclout/core/ and bitclout/backend/ directories to be in the same parent directory 
as mining-analysis. mining-analysis.go requires these libraries in order to construct, decode, and execute requests
 using the most recent production node specifications. The proper hierarchy can be made using the following bash snippet:
```shell
mkdir example
cd example/
git clone https://github.com/bitclout/backend
git clone https://github.com/bitclout/core
git clone https://github.com/bitclout/mining-analysis
```
## Example Execution
The following example uses the default flag arguments. The defaults will analyze the 1,000 most recent blocks using bitclout.com's API.
The output will be printed to the console.
```shell
go run mining_analysis.go
```
The following example is more complex, setting flags for the number of blocks to collect, which block height to start at,
as well as an output csv file for where to store the results.

To be explicit, the script will start at the block with height 40,000 and move backwards collecting the previous 2000 blocks.
The output will then be printed to both standard output and output.csv.
```shell
go run mining_analysis.go --blocks_to_collect=2000 --starting_block_height=40000 --output_csv_file=output.csv
```

## Example Output
The following shows example output for how the data will appear in a csv output file. This example uses 2,000 recently collected blocks.
```
BC1YLhGdJHfi6qWhuvxyQQqXU3dENb812oQC5PdLoyM6MQsdmScnTn2,425
BC1YLgANZLxhAKqPLAKt3P9ye1FVw6Lfa3gd4U4kYvEFTnJZauuPKg2,412
BC1YLjHcYoehKnXNXRKTGyyJzSmWKfzAuAin42iLwqduFjhasyPKCKs,328
BC1YLfvghL9UuYpaf8WBZFtCqn77rtLd1mr7md5qJykZb5RJMzgsGWs,290
BC1YLhHuq3UnGY6tJFvvquGeNPxfQqNTK8rZPieB7HKj7ivnsvktFaY,269
BC1YLi1pxsdYtfqJpS9HRQFn9gnRt3kqYKon9TAZ3YgcQRPK9XpRkyW,224
BC1YLhS3xZJ7RToJJc49dsmxBfABWkCax4r6eFbe5bSJJ63sH4p2yLN,25
BC1YLfjaLLCuz3fQxXGBqrwdBDYviSqpx4VuxJAaWzWHGHJW9wCEdbK,13
BC1YLfmBEnbGEnD7QuVvPv3uwuSqnRimkmjY9Ptf8fdXWLZeL2CNXUF,7
BC1YLiMvswR9N2fsGsGqHYyoq6w1iPUm7oFRy82UB2aH1FRtTA5HgMG,7
```

## Flags
| Flag      | Description |
| ----------- | ----------- |
| starting_block_height      | Specifies the block height to start at when collecting data. The script will step backwards from this starting block node to the node with height max(genesis, starting_block_height - blocks_to_collect) collecting data along the way. If not set, the script will start at the most recent tip.      |
| blocks_to_collect   | Specifies the number of blocks to collect moving backwards from starting_block_height. By default 1,000 blocks.       |
| output_csv_file | An optional csv output file where sorted data on public key -> # of blocks mined is stored. If the flag is not set, the data will only be outputed to the console. |
| node | The node from which to collect information on the blockchain. By default https://api.bitclout.com. |
| delay_milliseconds | The delay in milliseconds to wait between failed requests. By default 1,000ms. |