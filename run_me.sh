#!/usr/bin/env bash
set -e

# Init script to kick-start your project
url=$(git remote get-url origin)
gh_repo=${url#"git@github.com:"}
gh_repo=${gh_repo#"https://github.com/"}
gh_repo=${gh_repo%".git"}
copyright="$(date +%Y) $(git config user.name)"
project_name=$(basename "$gh_repo")
project_snake=$(echo "$project_name" | tr - _)
project_cap=$(echo "$project_name" | tr "[:lower:]" "[:upper:]" | tr - _)
project_words=$(echo "$project_name" | tr - ' ' | awk '{for(i=1;i<=NF;i++){ $i=toupper(substr($i,1,1)) substr($i,2) }}1')

echo "## Replacing all brick-starter-kit references by $project_name"
find . -name .git -prune -o -type f -not -name run_me.sh -print0 | xargs -0 perl -i -pe "s|bool64/brick-starter-kit|$gh_repo|g"
find . -name .git -prune -o -type f -not -name run_me.sh -print0 | xargs -0 perl -i -pe "s|brick-starter-kit|$project_name|g"
find . -name .git -prune -o -type f -not -name run_me.sh -print0 | xargs -0 perl -i -pe "s|brick_starter_kit|$project_snake|g"
find . -name .git -prune -o -type f -not -name run_me.sh -print0 | xargs -0 perl -i -pe "s|BRICK_STARTER_KIT|$project_cap|g"
find . -name .git -prune -o -type f -not -name run_me.sh -print0 | xargs -0 perl -i -pe "s|Brick Starter Kit|$project_words|g"

echo "## Removing this script"
rm ./run_me.sh

echo "## Please check the @TODO's:"
git grep TODO | grep -v run_me.sh

git add .

