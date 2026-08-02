// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { EIP712 } from "@openzeppelin/contracts/utils/cryptography/EIP712.sol";
import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { Ownable2Step } from "@openzeppelin/contracts/access/Ownable2Step.sol";
import { Pausable } from "@openzeppelin/contracts/utils/Pausable.sol";
import { ReentrancyGuard } from "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

contract MyferenceMarket is Ownable2Step, Pausable, ReentrancyGuard, EIP712 {
    using ECDSA for bytes32;
    using Math for uint256;

    error ZeroAmount();
    error InsufficientBalance();
    error InsufficientBond();
    error BondExitActive();
    error ExitDelayActive();
    error NothingToClaim();
    error TransferFailed();

    struct OfferVersion {
        bool active;
        bytes32 modelHash;
        bytes32 capabilityHash;
        uint64 version;
        uint256 inputPerMillion;
        uint256 outputPerMillion;
        uint256 computePerSecond;
    }

    address public immutable feeRecipient;
    address public immutable settlementSigner;
    uint256 public immutable minimumBond;
    uint64 public immutable bondExitDelay;
    uint64 public immutable feeDelay;

    mapping(address => uint256) public customerBalances;
    mapping(address => uint256) public claimable;
    mapping(address => uint256) public providerBonds;
    mapping(address => uint64) public bondExitAvailableAt;
    mapping(address => mapping(bytes32 => uint64)) public latestOfferVersion;
    mapping(address => mapping(bytes32 => mapping(uint64 => OfferVersion))) public offerVersions;

    uint256 public totalCustomerBalances;
    uint256 public totalProviderBonds;
    uint256 public totalLockedSessions;
    uint256 public totalClaimable;

    event Deposited(address indexed customer, uint256 amount);
    event WithdrawalRequested(address indexed account, uint256 amount);
    event Claimed(address indexed account, uint256 amount);
    event BondDeposited(address indexed provider, uint256 amount, uint256 totalBond);
    event BondExitRequested(address indexed provider, uint64 availableAt);
    event BondExitFinalized(address indexed provider, uint256 amount);
    event OfferPublished(
        address indexed provider,
        bytes32 indexed offerId,
        uint64 indexed version,
        bytes32 modelHash,
        bytes32 capabilityHash,
        uint256 inputPerMillion,
        uint256 outputPerMillion,
        uint256 computePerSecond
    );

    constructor(
        address initialOwner,
        address feeRecipient_,
        address settlementSigner_,
        uint256 minimumBond_,
        uint64 bondExitDelay_,
        uint64 feeDelay_
    ) Ownable(initialOwner) EIP712("MyferenceMarket", "1") {
        feeRecipient = feeRecipient_;
        settlementSigner = settlementSigner_;
        minimumBond = minimumBond_;
        bondExitDelay = bondExitDelay_;
        feeDelay = feeDelay_;
    }

    function deposit() external payable whenNotPaused {
        if (msg.value == 0) revert ZeroAmount();
        customerBalances[msg.sender] += msg.value;
        totalCustomerBalances += msg.value;
        emit Deposited(msg.sender, msg.value);
    }

    function requestWithdrawal(uint256 amount) external {
        if (amount == 0) revert ZeroAmount();
        if (amount > customerBalances[msg.sender]) revert InsufficientBalance();
        customerBalances[msg.sender] -= amount;
        totalCustomerBalances -= amount;
        _creditClaimable(msg.sender, amount);
        emit WithdrawalRequested(msg.sender, amount);
    }

    function claim() external nonReentrant {
        uint256 amount = claimable[msg.sender];
        if (amount == 0) revert NothingToClaim();
        claimable[msg.sender] = 0;
        totalClaimable -= amount;
        (bool sent,) = payable(msg.sender).call{ value: amount }("");
        if (!sent) revert TransferFailed();
        emit Claimed(msg.sender, amount);
    }

    function depositBond() external payable whenNotPaused {
        if (msg.value == 0) revert ZeroAmount();
        if (bondExitAvailableAt[msg.sender] != 0) revert BondExitActive();
        providerBonds[msg.sender] += msg.value;
        totalProviderBonds += msg.value;
        emit BondDeposited(msg.sender, msg.value, providerBonds[msg.sender]);
    }

    function requestBondExit() external {
        if (providerBonds[msg.sender] == 0) revert InsufficientBond();
        if (bondExitAvailableAt[msg.sender] != 0) revert BondExitActive();
        uint64 availableAt = uint64(block.timestamp) + bondExitDelay;
        bondExitAvailableAt[msg.sender] = availableAt;
        emit BondExitRequested(msg.sender, availableAt);
    }

    function finalizeBondExit() external {
        uint64 availableAt = bondExitAvailableAt[msg.sender];
        if (availableAt == 0 || block.timestamp < availableAt) revert ExitDelayActive();
        uint256 amount = providerBonds[msg.sender];
        providerBonds[msg.sender] = 0;
        bondExitAvailableAt[msg.sender] = 0;
        totalProviderBonds -= amount;
        _creditClaimable(msg.sender, amount);
        emit BondExitFinalized(msg.sender, amount);
    }

    function publishOffer(
        bytes32 offerId,
        bytes32 modelHash,
        bytes32 capabilityHash,
        uint256 inputPerMillion,
        uint256 outputPerMillion,
        uint256 computePerSecond
    ) external whenNotPaused {
        if (providerBonds[msg.sender] < minimumBond || bondExitAvailableAt[msg.sender] != 0) {
            revert InsufficientBond();
        }
        uint64 version = latestOfferVersion[msg.sender][offerId] + 1;
        latestOfferVersion[msg.sender][offerId] = version;
        offerVersions[msg.sender][offerId][version] = OfferVersion({
            active: true,
            modelHash: modelHash,
            capabilityHash: capabilityHash,
            version: version,
            inputPerMillion: inputPerMillion,
            outputPerMillion: outputPerMillion,
            computePerSecond: computePerSecond
        });
        emit OfferPublished(
            msg.sender,
            offerId,
            version,
            modelHash,
            capabilityHash,
            inputPerMillion,
            outputPerMillion,
            computePerSecond
        );
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }

    function _creditClaimable(address account, uint256 amount) internal {
        claimable[account] += amount;
        totalClaimable += amount;
    }
}
