<p align="center">
    <a href="https://cloud.ibm.com">
        <img src="https://cloud.ibm.com/media/docs/developer-appservice/resources/ibm-cloud.svg" height="100" alt="IBM Cloud">
    </a>
</p>


<p align="center">
    <a href="https://cloud.ibm.com">
    <img src="https://img.shields.io/badge/IBM%20Cloud-powered-blue.svg" alt="IBM Cloud">
    </a>
    <img src="https://img.shields.io/badge/platform-go-lightgrey.svg?style=flat" alt="platform">
    <img src="https://img.shields.io/badge/license-Apache2-blue.svg?style=flat" alt="Apache 2">
</p>


# IBM Cloud Starter Kit Operator for OpenShift (Alpha)

The _IBM Cloud Starter Kit Operator_ provides a simple OpenShift CRD-Based API that abstracts lower-level core Kubernetes and OpenShift APIs to simplify starter kit deployment. With this operator, you can set up a GitHub repository with the full application lifecycle (build and deploy) and consume IBM Cloud managed services with only one `oc apply`. The operator utilizes reconciliation logic to ensure that the required resources are automatically created and has a simple API interface that makes it easy to experiment with IBM Cloud starter kits.

IBM Starter Kits provide a quick starting point for various different application patterns that leverage IBM Cloud managed services and make it easy for developers to onboard their application workloads into IBM Cloud. A listing of curated IBM Cloud Starter Kits is available [here](https://github.com/search?q=topic%3Astarter-kit+org%3AIBM&type=Repositories).

> **Warning:** The current release is in an experimental alpha state and is still under active design and development. Do not use this in production environments.

## Pre-requisites

* Red Hat OpenShift Cluster
* It is recommended to install the [IBM Cloud Operator](https://operatorhub.io/operator/ibmcloud-operator) before installing this operator. You will need it to provision the `Service` and `Binding` CRDs that some of the examples reference.

## Installation

The _IBM Cloud Starter Kit Operator_ is available for installation via [OperatorHub](). Simply install the operator into your preferred namespace to access the `StarterKit` CRD.

### Setting up a GitHub access token as a Secret

Every starter kit that IBM offers is also a GitHub Template. The _IBM Cloud Starter Kit Operator_ uses a GitHub API token in order to create a repository from the selected GitHub Template and also automatically sets up a development webhook so changes to the repository automatically trigger a build and deployment to OpenShift.

First, you need to [create a Personal Access Token](https://github.com/settings/tokens) via GitHub with the **repo** and **admin:repo_hook** scopes enabled.

After creating a token, have your Administrator create a **Key/Value Secret** in OpenShift with the value retrieved from GitHub. Remember the **Secret Name** and **Key** because you will need to reference that later in the `StarterKit` CRD.

## Examples

This repository includes an [examples](./examples) folder with YAML examples of various starter kits available for deployment from within OpenShift.

You will need to update the replacement fields with your own values.

Some of the starter kits include the `Service` and `Binding` objects from the [IBM Cloud Operator](https://operatorhub.io/operator/ibmcloud-operator). If you have already deployed a `Service` or `Binding` object from within OpenShift, you can adjust the YAML to reference it (as-is, it will create a new instance and binding in addition to deploying the starter kit).

## How it works

Under the covers, the _IBM Cloud Starter Kit Operator_ does several things to speed up deployment to OpenShift:

* Creates a new GitHub repository from the referenced starter kit GitHub Template
* Creates and manages `Secret`, `Service`, `Route`, `ImageStream`, `BuildConfig`, and `DeploymentConfig` objects and sets the `StarterKit` as the owner.
* Automatically configures your `BuildConfig` with a webhook so changes to the created repository automatically kick off a build and deploy.
* Provides easy cleanup since the `StarterKit` owns all secondary resources. Simply execute `oc delete -f starter-kit.yaml` to clean up an instance and all of its managed resources.

> **Note:** The delete operation does not remove the associated GitHub repository that was created as part of the `StarterKit` instantiation process. This is considered to have a separate lifecycle than the in-cluster `StarterKit` instance, and this allows the user to continue to build out their application codebase as a separate artifact.

## License

This sample application is licensed under the Apache License, Version 2. Separate third-party code objects invoked within this code pattern are licensed by their respective providers pursuant to their own separate licenses. Contributions are subject to the [Developer Certificate of Origin, Version 1.1](https://developercertificate.org/) and the [Apache License, Version 2](https://www.apache.org/licenses/LICENSE-2.0.txt).

[Apache License FAQ](https://www.apache.org/foundation/license-faq.html#WhatDoesItMEAN)
