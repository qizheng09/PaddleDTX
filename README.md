[DOC](https://paddledtx.readthedocs.io) | [中文](./README_CN.md) | English

[![License](https://img.shields.io/badge/license-Apache%202-blue.svg)](LICENSE)

# PaddleDTX
PaddleDTX is a solution that focused on **distributed machine learning** technology based on **decentralized storage**. It solves the difficulties when massive private data needs to be securely stored and exchanged, also helps different parties break through isolated data islands to maximize the value of their data. 

## Overview of PaddleDTX
The computing layer of PaddleDTX is a network that composed of nodes of three kinds: **Requester**, **Executor** and **DataOwner**. The training samples and prediction dataset are stored in a decentralized storage network composed of DataOwner and **Storage** nodes. This decentralized storage network and the computing layer are supported by a underlying blockchain network.

### Secure Multi-party Computation Network
The Requester is a party with prediction demand, and the Executor is a party that is authorized by the DataOwner to gain access permit to the sample data for possible model training and result predicting. Multiple Executor nodes form an SMPC (secure multi-party computation) network. The Requester nodes publish the task to the blockchain network, and Executor nodes execute the task after authorization. The Executor nodes obtain sample data through the DataOwner, and the latter endorses the trust of data.

SMPC network is the framework that supports multiple distributed learning processes running in parallel. More vertical federated learning and horizontal federated learning algorithms will be supported in the future.

### Decentralized Storage Network
A DataOwner node processes its private data, and encryption, segmentation and replication related algorithms are used in this procedure, and finally encrypted fragments are distributed to multiple Storage nodes. A Storage node proves that it honestly holds the data fragments by answering the challenges generated by the DataOwner. Through these mechanisms, storage resources can be safely maintained without violating any data privacy. Please refer to [XuperDB](./xdb/README.md) for more about design principle and implementation. 

### Blockchain Network
Training tasks and prediction tasks will be broadcasted to the Executor nodes by a blockchain network. Then the Executor nodes involved will execute these tasks. The DataOwner node and the Storage node exchange information through the blockchain network when monitoring files and nodes health status, and also in the challenge-answer-verify process of replicas holding proof.

Currently, XuperChain is the only blockchain framework that PaddleDTX supported.

![Image text](./images/architecture.png)

## Vertical Federated Learning
The open source version of PaddleDTX supports two-party vertical federated learning(VFL) algorithms, including Linear Regression and Logistic Regression, more algorithms such as two-party Neural Network will be open sourced soon, along with multi-party VFL and multi-party HFL(horizontal federated learning) algorithms. Please refer to [crypto/ml](./crypto/core/machine_learning) for more about background and implementation of these two algorithms. 

Training and predicting steps of VFL are shown as follows:

![Image text](./images/vertical_learning.png)

### Sample Preparation
A FL task needs to specify sample files that will be used in computation or prediction, and these files are stored in the decentralized storage system(XuperDB). Before executing a task, executor(often data owner) needs to fetch its own sample files from XuperDB.

### Sample Alignment
Both VFL training and prediction tasks require a sample alignment process. That is, to find sample intersections by using all the participants' ID lists. Training and predicting are performed on intersected samples.

The project implemented PSI(Private Set Intersection) for sample alignment without leaking any participant's ID. Refer to [crypto/psi](./crypto/core/machine_learning/linear_regression/gradient_descent/mpc_vertical/psi.go) for more details about PSI.

### Training Process
Model training is an iterated process, which relies on collaborative computing of two parities' samples. Participants need to exchange intermediate parameters during many training epochs, in order to get proper local model for each party.

To ensure confidentiality of each participant's data, Paillier cryptosystem is used for parameters encryption and decryption. Paillier is an additive homomorphic algorithm, which enables us to do addition or scalar multiplication on ciphertext directly. Refer to [crypto/paillier](./crypto/common/math/homomorphism/paillier/paillier.go) for more details about Paillier.

### Prediction Process
Prediction task requires a model, so related training task needs to be done before prediction task starts. Models are separately stored in participants' local storage. Participants compute local prediction result using their own model, and then gather all partial prediction results to deduce final result.

For linear regression, destandardization process can be performed after gathering all partial results. This process is only able to be done by the party has labels. So all partial results will be sent to the party has labels, which will deduce final result and store it as a file in XuperDB for requester to use.

## Installation
There are two ways of installing PaddleDTX:

### Run PaddleDTX in docker
We **highly recommend** to run PaddleDTX in Docker.
You could install all the components with docker images provided by us. Please refer to [starting network](./testdata/README.md). If you want to build docker images locally, please refer to [building image of PaddleDTX](./dai/build_image.sh) and [building image of XuperDB](./xdb/build_image.sh).

### Install PaddleDTX from source code
To build PaddleDTX from source code, you need:

* go 1.13 or greater
```sh
# In dai directory
make

# In xdb directory 
make
```
You could get installation package from `./output` and install it manually.

## Testing
We provide [test scripts](./scripts/README.md) for you to test, understand and use PaddleDTX.


## Related Work
[1] Konečný J, McMahan H B, Yu F X, et al. Federated learning: Strategies for improving communication efficiency[J]. arXiv preprint arXiv:1610.05492, 2016.

[2] Yang Q, Liu Y, Chen T, et al. Federated machine learning: Concept and applications[J]. ACM Transactions on Intelligent Systems and Technology (TIST), 2019, 10(2): 1-19.

[3] Goodfellow I, Bengio Y, Courville A. Deep learning[M]. MIT press, 2016.

[4] Goodfellow I, Bengio Y, Courville A. Machine learning basics[J]. Deep learning, 2016, 1(7): 98-164.

[5] Paillier P. Public-key cryptosystems based on composite degree residuosity classes[C]//International conference on the theory and applications of cryptographic techniques. Springer, Berlin, Heidelberg, 1999: 223-238.

[6] Lo H K. Insecurity of quantum secure computations[J]. Physical Review A, 1997, 56(2): 1154.

[7] Chen H, Laine K, Rindal P. Fast private set intersection from homomorphic encryption[C]//Proceedings of the 2017 ACM SIGSAC Conference on Computer and Communications Security. 2017: 1243-1255.

[8] Shamir A. How to share a secret[J]. Communications of the ACM, 1979, 22(11): 612-613.

[9] https://xuper.baidu.com/n/xuperdoc/general_introduction/brief.html
