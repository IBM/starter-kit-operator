
IMAGE=ibmcom/starter-kit-operator
# update FORK to override default value
FORK=${FORK:=https://github.com/nfstein/community-operators}

echo Image tags in remote repository $IMAGE:
wget -q https://registry.hub.docker.com/v1/repositories/$IMAGE/tags -O - | jq -c .[].name

read -p "Enter tag for image $IMAGE (no quotes) e.g. 0.0.0 : " TAG

GITHUB_WORKSPACE=$(pwd)

rm -rf $GITHUB_WORKSPACE/community-operators-for-fork

mkdir community-operators-for-fork
git clone https://github.com/operator-framework/community-operators/ community-operators-for-fork
cd community-operators-for-fork/community-operators/starter-kit-operator
ls

echo The script uses previous versions of the operator to build the new version
read -p "Enter operatorhub version of OLD operator from the list above e.g 0.0.0 : " OLD_OPERATOR_VERSION
read -p "Enter operatorhub version for NEW operator e.g 0.0.1 : " NEW_OPERATOR_VERSION

rm -rf $NEW_OPERATOR_VERSION
cp -r $OLD_OPERATOR_VERSION $NEW_OPERATOR_VERSION

mv $NEW_OPERATOR_VERSION/starter-kit-operator.v$OLD_OPERATOR_VERSION.clusterserviceversion.yaml $NEW_OPERATOR_VERSION/starter-kit-operator.v$NEW_OPERATOR_VERSION.clusterserviceversion.yaml

yq write --inplace starter-kit-operator.package.yaml channels[0].currentCSV starter-kit-operator.v$NEW_OPERATOR_VERSION

SKIT_CSV=$NEW_OPERATOR_VERSION/starter-kit-operator.v$NEW_OPERATOR_VERSION.clusterserviceversion.yaml

echo Updating YAMLS
yq write --inplace $SKIT_CSV metadata.annotations.containerImage $IMAGE:$OPERATOR_IMAGE_TAG
yq write --inplace $SKIT_CSV metadata.name starter-kit-operator.v$NEW_OPERATOR_VERSION
yq write --inplace $SKIT_CSV spec.install.spec.deployments[0].spec.template.spec.containers[0].image $IMAGE:$OPERATOR_IMAGE_TAG
yq write --inplace $SKIT_CSV spec.replaces starter-kit-operator.v$OLD_OPERATOR_VERSION
yq write --inplace $SKIT_CSV spec.version $NEW_OPERATOR_VERSION

echo Performing git actions
git add .
git commit -m "Update IBM Starter Kit operator to v$NEW_OPERATOR_VERSION"

git checkout -b create-$NEW_OPERATOR_VERSION

git remote add fork $FORK

echo Check out the contents of the new version at $(pwd)/$NEW_OPERATOR_VERSION
echo The new version needs to be pushed to a fork of operator-framework/community-operators
echo to create the pr to update the version. You can make changes before you push.
echo note: you can set the FORK variable before running the script to push to your own fork
read -p "Push contents to $FORK? y/n : " uservar
if [ "$uservar" == "y" ]; then
	set -x
	git push --set-upstream fork create-$NEW_OPERATOR_VERSION
	set +x
	cd -
	echo Cleaning up
	rm -rf $GITHUB_WORKSPACE/community-operators-for-fork
else
	echo When you\'re ready to push, run "git push --set-upstream fork create-$NEW_OPERATOR_VERSION"
fi

echo Open a PR against the parent repo here: https://github.com/operator-framework/community-operators/pulls
