// Copyright 2018-2025 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package conn

// AWS Partition identifiers
const (
	// PartitionAWS is the standard AWS commercial partition
	PartitionAWS = "aws"
	
	// PartitionAWSCN is the AWS China partition
	PartitionAWSCN = "aws-cn"
	
	// PartitionAWSUSGov is the AWS GovCloud (US) partition
	PartitionAWSUSGov = "aws-us-gov"
	
	// PartitionAWSISO is the AWS ISO (US) partition
	PartitionAWSISO = "aws-iso"
	
	// PartitionAWSISOB is the AWS ISO-B (US) partition
	PartitionAWSISOB = "aws-iso-b"
)

// AWS domain suffixes for different partitions
const (
	// DomainSuffixAWS is the domain suffix for standard AWS regions
	DomainSuffixAWS = "amazonaws.com"
	
	// DomainSuffixAWSCN is the domain suffix for AWS China regions
	DomainSuffixAWSCN = "amazonaws.com.cn"
	
	// DomainSuffixAWSISO is the domain suffix for AWS ISO regions
	DomainSuffixAWSISO = "c2s.ic.gov"
	
	// DomainSuffixAWSISOB is the domain suffix for AWS ISO-B regions
	DomainSuffixAWSISOB = "sc2s.sgov.gov"
)