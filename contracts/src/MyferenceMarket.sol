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
    error SessionAlreadyExists();
    error InvalidSession();
    error SessionExpired();
    error SessionCloseDelayActive();
    error SessionAllowanceExceeded();
    error RequestAlreadySettled();
    error NonceAlreadyUsed();
    error InvalidReceipt();
    error InvalidReceiptSignature();
    error StaleOffer();
    error MaximumChargeExceeded();
    error InvalidBatch();
    error InvalidEvidence();
    error EvidenceAlreadyUsed();
    error FeeTooHigh();
    error FeeDelayActive();
    error NoPendingFee();

    struct OfferVersion {
        bool active;
        bytes32 modelHash;
        bytes32 capabilityHash;
        uint64 version;
        uint256 inputPerMillion;
        uint256 outputPerMillion;
        uint256 computePerSecond;
    }

    struct Session {
        address customer;
        uint256 allowance;
        uint256 spent;
        uint64 expiresAt;
        uint64 closeAvailableAt;
        bool finalized;
    }

    struct Receipt {
        bytes32 requestId;
        bytes32 sessionId;
        address customer;
        address provider;
        address settlementSigner;
        bytes32 offerId;
        uint64 priceVersion;
        bytes32 modelHash;
        bytes32 capabilityHash;
        uint64 inputTokens;
        uint64 outputTokens;
        uint64 computeMilliseconds;
        uint64 maximumCharge;
        uint64 totalCharge;
        uint16 feeBasisPoints;
        uint64 feeVersion;
        uint8 status;
        uint64 completedAt;
        bytes32 inputHash;
        bytes32 outputHash;
        uint64 nonce;
    }

    bytes32 public constant RECEIPT_TYPEHASH = keccak256(
        "Receipt(bytes32 requestId,bytes32 sessionId,address customer,address provider,address settlementSigner,bytes32 offerId,uint64 priceVersion,bytes32 modelHash,bytes32 capabilityHash,uint64 inputTokens,uint64 outputTokens,uint64 computeMilliseconds,uint64 maximumCharge,uint64 totalCharge,uint16 feeBasisPoints,uint64 feeVersion,uint8 status,uint64 completedAt,bytes32 inputHash,bytes32 outputHash,uint64 nonce)"
    );
    uint64 public constant SESSION_CLOSE_DELAY = 1 days;
    uint16 public constant MAXIMUM_FEE_BASIS_POINTS = 1_500;

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
    mapping(bytes32 => Session) public sessions;
    mapping(bytes32 => bool) public settledRequests;
    mapping(address => mapping(uint64 => bool)) public usedNonces;
    mapping(bytes32 => bool) public slashedRequests;
    mapping(uint64 => uint16) public feeBpsByVersion;

    uint16 public feeBasisPoints = 500;
    uint64 public feeVersion = 1;
    uint16 public pendingFeeBasisPoints;
    uint64 public pendingFeeAvailableAt;
    bool public pendingFeeChange;

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
    event SessionOpened(
        bytes32 indexed sessionId, address indexed customer, uint256 allowance, uint64 expiresAt
    );
    event SessionCloseRequested(bytes32 indexed sessionId, uint64 availableAt);
    event SessionClosed(bytes32 indexed sessionId, uint256 returnedAmount);
    event ReceiptSettled(
        bytes32 indexed requestId,
        bytes32 indexed sessionId,
        address indexed provider,
        uint256 providerAmount,
        uint256 feeAmount
    );
    event ProviderSlashed(address indexed provider, bytes32 indexed requestId, uint256 amount);
    event FeeProposed(uint16 feeBasisPoints, uint64 availableAt);
    event FeeChanged(uint16 feeBasisPoints, uint64 version);

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
        feeBpsByVersion[1] = 500;
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

    function openSession(bytes32 sessionId, uint256 allowance, uint64 expiresAt)
        external
        whenNotPaused
    {
        if (sessionId == bytes32(0) || allowance == 0 || expiresAt <= block.timestamp) {
            revert InvalidSession();
        }
        if (sessions[sessionId].customer != address(0)) revert SessionAlreadyExists();
        if (customerBalances[msg.sender] < allowance) revert InsufficientBalance();
        customerBalances[msg.sender] -= allowance;
        totalCustomerBalances -= allowance;
        totalLockedSessions += allowance;
        sessions[sessionId] = Session({
            customer: msg.sender,
            allowance: allowance,
            spent: 0,
            expiresAt: expiresAt,
            closeAvailableAt: 0,
            finalized: false
        });
        emit SessionOpened(sessionId, msg.sender, allowance, expiresAt);
    }

    function requestSessionClose(bytes32 sessionId) external {
        Session storage session = sessions[sessionId];
        if (session.customer != msg.sender || session.finalized) revert InvalidSession();
        if (session.closeAvailableAt != 0) revert SessionCloseDelayActive();
        uint64 availableAt = uint64(block.timestamp) + SESSION_CLOSE_DELAY;
        session.closeAvailableAt = availableAt;
        emit SessionCloseRequested(sessionId, availableAt);
    }

    function finalizeSessionClose(bytes32 sessionId) external {
        Session storage session = sessions[sessionId];
        if (session.customer != msg.sender || session.finalized) revert InvalidSession();
        if (session.closeAvailableAt == 0 || block.timestamp < session.closeAvailableAt) {
            revert SessionCloseDelayActive();
        }
        session.finalized = true;
        uint256 remaining = session.allowance - session.spent;
        totalLockedSessions -= remaining;
        customerBalances[msg.sender] += remaining;
        totalCustomerBalances += remaining;
        emit SessionClosed(sessionId, remaining);
    }

    function hashReceipt(Receipt memory receipt) public view returns (bytes32) {
        return _hashTypedDataV4(keccak256(abi.encode(RECEIPT_TYPEHASH, receipt)));
    }

    function settleReceipt(
        Receipt calldata receipt,
        bytes calldata providerSignature,
        bytes calldata settlementSignature
    ) external whenNotPaused {
        _settleReceipt(receipt, providerSignature, settlementSignature);
    }

    function settleReceipts(
        Receipt[] calldata receipts,
        bytes[] calldata providerSignatures,
        bytes[] calldata settlementSignatures
    ) external whenNotPaused {
        if (
            receipts.length == 0 || receipts.length != providerSignatures.length
                || receipts.length != settlementSignatures.length
        ) revert InvalidBatch();
        for (uint256 i; i < receipts.length; ++i) {
            _settleReceipt(receipts[i], providerSignatures[i], settlementSignatures[i]);
        }
    }

    function slashDoubleSign(
        Receipt calldata first,
        bytes calldata firstSignature,
        Receipt calldata second,
        bytes calldata secondSignature
    ) external {
        bytes32 firstDigest = hashReceipt(first);
        bytes32 secondDigest = hashReceipt(second);
        if (
            first.requestId == bytes32(0) || first.requestId != second.requestId
                || first.provider == address(0) || first.provider != second.provider
                || firstDigest == secondDigest
        ) revert InvalidEvidence();
        if (slashedRequests[first.requestId]) revert EvidenceAlreadyUsed();
        if (
            firstDigest.recover(firstSignature) != first.provider
                || secondDigest.recover(secondSignature) != first.provider
        ) revert InvalidReceiptSignature();

        uint256 amount = Math.min(providerBonds[first.provider], minimumBond);
        if (amount == 0) revert InsufficientBond();
        slashedRequests[first.requestId] = true;
        providerBonds[first.provider] -= amount;
        totalProviderBonds -= amount;
        if (providerBonds[first.provider] == 0) bondExitAvailableAt[first.provider] = 0;
        _creditClaimable(feeRecipient, amount);
        emit ProviderSlashed(first.provider, first.requestId, amount);
    }

    function proposeFee(uint16 newFeeBasisPoints) external onlyOwner {
        if (newFeeBasisPoints > MAXIMUM_FEE_BASIS_POINTS) revert FeeTooHigh();
        pendingFeeBasisPoints = newFeeBasisPoints;
        pendingFeeAvailableAt = uint64(block.timestamp) + feeDelay;
        pendingFeeChange = true;
        emit FeeProposed(newFeeBasisPoints, pendingFeeAvailableAt);
    }

    function executeFeeChange() external {
        if (!pendingFeeChange) revert NoPendingFee();
        if (block.timestamp < pendingFeeAvailableAt) revert FeeDelayActive();
        feeBasisPoints = pendingFeeBasisPoints;
        ++feeVersion;
        feeBpsByVersion[feeVersion] = feeBasisPoints;
        pendingFeeBasisPoints = 0;
        pendingFeeAvailableAt = 0;
        pendingFeeChange = false;
        emit FeeChanged(feeBasisPoints, feeVersion);
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

    function _settleReceipt(
        Receipt calldata receipt,
        bytes calldata providerSignature,
        bytes calldata settlementSignature
    ) internal {
        if (settledRequests[receipt.requestId]) {
            revert RequestAlreadySettled();
        }
        if (usedNonces[receipt.provider][receipt.nonce]) revert NonceAlreadyUsed();
        if (
            receipt.requestId == bytes32(0) || receipt.status != 1
                || receipt.settlementSigner != settlementSigner || receipt.feeVersion == 0
                || feeBpsByVersion[receipt.feeVersion] != receipt.feeBasisPoints
                || receipt.completedAt > block.timestamp
        ) revert InvalidReceipt();

        Session storage session = sessions[receipt.sessionId];
        if (
            session.customer != receipt.customer || session.customer == address(0)
                || session.finalized
        ) revert InvalidSession();
        if (receipt.completedAt > session.expiresAt) revert SessionExpired();

        OfferVersion storage offer =
            offerVersions[receipt.provider][receipt.offerId][receipt.priceVersion];
        if (
            !offer.active || offer.modelHash != receipt.modelHash
                || offer.capabilityHash != receipt.capabilityHash
        ) revert StaleOffer();

        uint256 charge = Math.mulDiv(
            receipt.inputTokens, offer.inputPerMillion, 1_000_000, Math.Rounding.Ceil
        ) + Math.mulDiv(receipt.outputTokens, offer.outputPerMillion, 1_000_000, Math.Rounding.Ceil)
        + Math.mulDiv(
            receipt.computeMilliseconds, offer.computePerSecond, 1_000, Math.Rounding.Ceil
        );
        if (charge != receipt.totalCharge) revert InvalidReceipt();
        if (charge > receipt.maximumCharge) revert MaximumChargeExceeded();
        if (session.spent + charge > session.allowance) revert SessionAllowanceExceeded();

        bytes32 digest = hashReceipt(receipt);
        if (
            digest.recover(providerSignature) != receipt.provider
                || digest.recover(settlementSignature) != settlementSigner
        ) revert InvalidReceiptSignature();

        settledRequests[receipt.requestId] = true;
        usedNonces[receipt.provider][receipt.nonce] = true;
        session.spent += charge;
        totalLockedSessions -= charge;

        uint256 feeAmount = Math.mulDiv(charge, receipt.feeBasisPoints, 10_000);
        uint256 providerAmount = charge - feeAmount;
        _creditClaimable(receipt.provider, providerAmount);
        _creditClaimable(feeRecipient, feeAmount);
        emit ReceiptSettled(
            receipt.requestId, receipt.sessionId, receipt.provider, providerAmount, feeAmount
        );
    }
}
