#!/usr/bin/env bash
DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$DIR" ]]; then DIR="$PWD"; fi
. "$DIR/common.sh"

alias="chain1"

wasp-cli init
wasp-cli request-funds

r=$(wasp-cli address)
echo "$r"
[[ "$r" =~ Address:[[:space:]]+([[:alnum:]]+)$ ]]
owneraddr=${BASH_REMATCH[1]}

[[ $(wasp-cli chain list | tail -n +5 | wc -l) == "0" ]]

# deploy a chain
wasp-cli chain deploy --chain=$alias --committee='0,1,2,3' --quorum=3
chainid=$(cat wasp-cli.json | jq .chains.$alias -r)

r=$(wasp-cli chain list)
[[ $(echo "$r" | tail -n +5 | wc -l) == "1" ]]
[[ "$r" =~ "$chainid" ]]

# unnecessary, since it is the latest deployed chain
wasp-cli set chain $alias

# test chain info command
r=$(wasp-cli chain info)
echo "$r"
# test that the chainid is shown
[[ "$r" =~ "$chainid" ]]

# test the list-contracts command
r=$(wasp-cli chain list-contracts)
echo "$r"
# check that all core contracts are listed
[[ $(echo "$r" | tail -n +5 | wc -l) == "4" ]]

# test the list-accounts command
r=$(wasp-cli chain list-accounts)
echo "$r"
# check that the owner is listed
echo "$r" | grep -q "$owneraddr"

agentid=$(echo "$r" | grep "$owneraddr" | sed 's/[:[:space:]].*$//')

r=$(wasp-cli chain balance "$agentid")
echo "$r"
# check that the chain balance of owner is 1 IOTA
[[ $(echo "$r" | tail -n +4) =~ IOTA\ +1\ *$ ]]

# same test, this time calling the view function manually
r=$(wasp-cli chain call-view accounts balance string a agentid "$agentid" | wasp-cli decode color int)
[[ "$r" == "IOTA: 1" ]]

# test the chainlog
r=$(wasp-cli chain log root)
echo "$r"
[[ $(echo "$r" | wc -l) == "1" ]]

echo "PASS"
