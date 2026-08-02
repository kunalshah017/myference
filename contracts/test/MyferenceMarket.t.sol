// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

import { Test } from "forge-std/Test.sol";
import { MyferenceMarket } from "../src/MyferenceMarket.sol";

contract MyferenceMarketTest is Test {
    MyferenceMarket internal market;

    address internal owner = makeAddr("owner");
    address internal customer = makeAddr("customer");
    uint256 internal providerKey = 0xA11CE;
    uint256 internal settlementKey = 0xB0B;
    address internal provider;
    address internal feeRecipient = makeAddr("feeRecipient");
    address internal settlementSigner;

    uint256 internal constant MINIMUM_BOND = 10 ether;
    uint64 internal constant EXIT_DELAY = 7 days;
    bytes32 internal constant OFFER_ID = keccak256("ollama/qwen3:8b");
    bytes32 internal constant MODEL_HASH = keccak256("qwen3:8b");
    bytes32 internal constant CAPABILITY_HASH = keccak256("chat-completions");
    bytes32 internal constant SESSION_ID = keccak256("session-1");

    function setUp() public {
        provider = vm.addr(providerKey);
        settlementSigner = vm.addr(settlementKey);
        market = new MyferenceMarket(
            owner, feeRecipient, settlementSigner, MINIMUM_BOND, EXIT_DELAY, 2 days
        );
        vm.deal(customer, 100 ether);
        vm.deal(provider, 100 ether);
    }

    function testDepositCreditsOnlySender() public {
        vm.prank(customer);
        market.deposit{ value: 3 ether }();

        assertEq(market.customerBalances(customer), 3 ether);
        assertEq(market.customerBalances(provider), 0);
        assertEq(address(market).balance, 3 ether);
    }

    function testWithdrawalMovesFundsToPullPaymentBeforeClaim() public {
        vm.startPrank(customer);
        market.deposit{ value: 3 ether }();
        market.requestWithdrawal(2 ether);

        assertEq(market.customerBalances(customer), 1 ether);
        assertEq(market.claimable(customer), 2 ether);

        uint256 beforeClaim = customer.balance;
        market.claim();
        vm.stopPrank();

        assertEq(customer.balance, beforeClaim + 2 ether);
        assertEq(market.claimable(customer), 0);
    }

    function testUnbondedProviderCannotPublish() public {
        vm.prank(provider);
        vm.expectRevert(MyferenceMarket.InsufficientBond.selector);
        market.publishOffer(OFFER_ID, MODEL_HASH, CAPABILITY_HASH, 10, 20, 30);
    }

    function testBondedProviderPublishesImmutablePriceVersions() public {
        vm.startPrank(provider);
        market.depositBond{ value: MINIMUM_BOND }();
        market.publishOffer(OFFER_ID, MODEL_HASH, CAPABILITY_HASH, 10, 20, 30);
        market.publishOffer(OFFER_ID, MODEL_HASH, CAPABILITY_HASH, 11, 22, 33);
        vm.stopPrank();

        assertEq(market.latestOfferVersion(provider, OFFER_ID), 2);
        (,,,, uint256 inputRateV1, uint256 outputRateV1, uint256 computeRateV1) =
            market.offerVersions(provider, OFFER_ID, 1);
        (,,,, uint256 inputRateV2, uint256 outputRateV2, uint256 computeRateV2) =
            market.offerVersions(provider, OFFER_ID, 2);
        assertEq(inputRateV1, 10);
        assertEq(outputRateV1, 20);
        assertEq(computeRateV1, 30);
        assertEq(inputRateV2, 11);
        assertEq(outputRateV2, 22);
        assertEq(computeRateV2, 33);
    }

    function testBondCannotExitBeforeDelay() public {
        vm.startPrank(provider);
        market.depositBond{ value: MINIMUM_BOND }();
        market.requestBondExit();
        vm.expectRevert(MyferenceMarket.ExitDelayActive.selector);
        market.finalizeBondExit();
        vm.warp(block.timestamp + EXIT_DELAY);
        market.finalizeBondExit();
        vm.stopPrank();

        assertEq(market.providerBonds(provider), 0);
        assertEq(market.claimable(provider), MINIMUM_BOND);
    }

    function testFuzzDepositAndWithdrawalConserveMon(uint128 rawAmount, uint128 rawWithdrawal)
        public
    {
        uint256 amount = bound(uint256(rawAmount), 1, 100 ether);
        uint256 withdrawal = bound(uint256(rawWithdrawal), 1, amount);
        vm.deal(customer, amount);

        vm.startPrank(customer);
        market.deposit{ value: amount }();
        market.requestWithdrawal(withdrawal);
        vm.stopPrank();

        assertEq(
            address(market).balance,
            market.totalCustomerBalances() + market.totalProviderBonds()
                + market.totalLockedSessions() + market.totalClaimable()
        );
    }

    function testSessionLocksAllowanceAndReturnsOnlyUnusedFundsAfterCloseDelay() public {
        vm.startPrank(customer);
        market.deposit{ value: 5 ether }();
        market.openSession(SESSION_ID, 3 ether, uint64(block.timestamp + 2 days));
        market.requestSessionClose(SESSION_ID);

        assertEq(market.customerBalances(customer), 2 ether);
        assertEq(market.totalLockedSessions(), 3 ether);
        vm.expectRevert(MyferenceMarket.SessionCloseDelayActive.selector);
        market.finalizeSessionClose(SESSION_ID);
        vm.warp(block.timestamp + market.SESSION_CLOSE_DELAY());
        market.finalizeSessionClose(SESSION_ID);
        vm.stopPrank();

        assertEq(market.customerBalances(customer), 5 ether);
        assertEq(market.totalLockedSessions(), 0);
    }

    function testSettleReceiptUsesExactBillingAndDistributesNinetyFiveFive() public {
        _bondPublish(1 ether, 0, 0);
        _openSession(2 ether, uint64(block.timestamp + 2 days));
        MyferenceMarket.Receipt memory receipt = _receipt(keccak256("request-1"), 1, 1 ether);

        market.settleReceipt(receipt, _sign(providerKey, receipt), _sign(settlementKey, receipt));

        assertEq(market.claimable(provider), 0.95 ether);
        assertEq(market.claimable(feeRecipient), 0.05 ether);
        (,, uint256 spent,,,) = market.sessions(SESSION_ID);
        assertEq(spent, 1 ether);
        assertTrue(market.settledRequests(receipt.requestId));
    }

    function testSettleRejectsReplayBadSignatureStalePriceAndMaximumExceeded() public {
        _bondPublish(1 ether, 0, 0);
        _openSession(3 ether, uint64(block.timestamp + 2 days));

        MyferenceMarket.Receipt memory valid = _receipt(keccak256("request-valid"), 1, 1 ether);
        market.settleReceipt(valid, _sign(providerKey, valid), _sign(settlementKey, valid));
        _expectSettleRevert(
            MyferenceMarket.RequestAlreadySettled.selector, valid, providerKey, settlementKey
        );

        MyferenceMarket.Receipt memory badSignature =
            _receipt(keccak256("request-bad-signature"), 2, 1 ether);
        _expectSettleRevert(
            MyferenceMarket.InvalidReceiptSignature.selector,
            badSignature,
            settlementKey,
            settlementKey
        );

        MyferenceMarket.Receipt memory stale = _receipt(keccak256("request-stale"), 3, 1 ether);
        stale.priceVersion = 2;
        _expectSettleRevert(MyferenceMarket.StaleOffer.selector, stale, providerKey, settlementKey);

        MyferenceMarket.Receipt memory overMaximum =
            _receipt(keccak256("request-over-maximum"), 4, 0.5 ether);
        _expectSettleRevert(
            MyferenceMarket.MaximumChargeExceeded.selector, overMaximum, providerKey, settlementKey
        );
    }

    function testSessionExpiryAndAllowanceAreEnforcedWhileClosingAllowsPendingSettlement() public {
        _bondPublish(1 ether, 0, 0);
        uint64 expiry = uint64(block.timestamp + 1 days);
        _openSession(1 ether, expiry);
        vm.prank(customer);
        market.requestSessionClose(SESSION_ID);

        MyferenceMarket.Receipt memory pending = _receipt(keccak256("request-pending"), 1, 1 ether);
        market.settleReceipt(pending, _sign(providerKey, pending), _sign(settlementKey, pending));

        MyferenceMarket.Receipt memory exhausted =
            _receipt(keccak256("request-exhausted"), 2, 1 ether);
        _expectSettleRevert(
            MyferenceMarket.SessionAllowanceExceeded.selector, exhausted, providerKey, settlementKey
        );

        bytes32 expiredSession = keccak256("expired-session");
        vm.startPrank(customer);
        market.deposit{ value: 1 ether }();
        market.openSession(expiredSession, 1 ether, uint64(block.timestamp + 1));
        vm.stopPrank();
        vm.warp(block.timestamp + 2);
        MyferenceMarket.Receipt memory expired = _receipt(keccak256("request-expired"), 3, 1 ether);
        expired.sessionId = expiredSession;
        expired.completedAt = uint64(block.timestamp);
        _expectSettleRevert(
            MyferenceMarket.SessionExpired.selector, expired, providerKey, settlementKey
        );
    }

    function testSettleBatchIsAtomic() public {
        _bondPublish(1 ether, 0, 0);
        _openSession(2 ether, uint64(block.timestamp + 2 days));
        MyferenceMarket.Receipt[] memory receipts = new MyferenceMarket.Receipt[](2);
        bytes[] memory providerSignatures = new bytes[](2);
        bytes[] memory settlementSignatures = new bytes[](2);
        receipts[0] = _receipt(keccak256("batch-1"), 1, 1 ether);
        receipts[1] = _receipt(keccak256("batch-2"), 2, 1 ether);
        providerSignatures[0] = _sign(providerKey, receipts[0]);
        providerSignatures[1] = _sign(providerKey, receipts[1]);
        settlementSignatures[0] = _sign(settlementKey, receipts[0]);
        settlementSignatures[1] = _sign(providerKey, receipts[1]);

        vm.expectRevert(MyferenceMarket.InvalidReceiptSignature.selector);
        market.settleReceipts(receipts, providerSignatures, settlementSignatures);

        assertFalse(market.settledRequests(receipts[0].requestId));
        assertEq(market.claimable(provider), 0);
    }

    function testSlashConflictingProviderReceiptsExactlyOnce() public {
        _bondPublish(1 ether, 0, 0);
        MyferenceMarket.Receipt memory first = _receipt(keccak256("equivocation"), 1, 1 ether);
        MyferenceMarket.Receipt memory second = _receipt(keccak256("equivocation"), 1, 1 ether);
        second.outputHash = keccak256("conflicting-output");
        bytes memory firstSignature = _sign(providerKey, first);
        bytes memory secondSignature = _sign(providerKey, second);

        market.slashDoubleSign(first, firstSignature, second, secondSignature);

        assertEq(market.providerBonds(provider), 0);
        assertEq(market.claimable(feeRecipient), MINIMUM_BOND);
        assertTrue(market.slashedRequests(first.requestId));
        vm.expectRevert(MyferenceMarket.EvidenceAlreadyUsed.selector);
        market.slashDoubleSign(first, firstSignature, second, secondSignature);
    }

    function testSlashRejectsIdenticalOrArbitraryEvidence() public {
        _bondPublish(1 ether, 0, 0);
        MyferenceMarket.Receipt memory first = _receipt(keccak256("same"), 1, 1 ether);
        bytes memory signature = _sign(providerKey, first);
        vm.expectRevert(MyferenceMarket.InvalidEvidence.selector);
        market.slashDoubleSign(first, signature, first, signature);

        MyferenceMarket.Receipt memory second = _receipt(keccak256("same"), 1, 1 ether);
        second.outputHash = keccak256("different");
        bytes memory arbitrarySignature = _sign(settlementKey, second);
        vm.expectRevert(MyferenceMarket.InvalidReceiptSignature.selector);
        market.slashDoubleSign(first, signature, second, arbitrarySignature);
    }

    function testFeeChangeIsCappedVersionedAndTimelocked() public {
        vm.prank(owner);
        market.proposeFee(600);
        vm.expectRevert(MyferenceMarket.FeeDelayActive.selector);
        market.executeFeeChange();

        vm.warp(block.timestamp + 2 days);
        market.executeFeeChange();
        assertEq(market.feeBasisPoints(), 600);
        assertEq(market.feeVersion(), 2);
        assertEq(market.feeBpsByVersion(1), 500);
        assertEq(market.feeBpsByVersion(2), 600);

        vm.prank(owner);
        vm.expectRevert(MyferenceMarket.FeeTooHigh.selector);
        market.proposeFee(1_501);
    }

    function testPauseCannotBlockMatureWithdrawals() public {
        vm.prank(customer);
        market.deposit{ value: 2 ether }();
        vm.prank(provider);
        market.depositBond{ value: MINIMUM_BOND }();
        vm.prank(provider);
        market.requestBondExit();
        vm.prank(owner);
        market.pause();

        vm.prank(customer);
        market.requestWithdrawal(1 ether);
        vm.warp(block.timestamp + EXIT_DELAY);
        vm.prank(provider);
        market.finalizeBondExit();

        assertEq(market.claimable(customer), 1 ether);
        assertEq(market.claimable(provider), MINIMUM_BOND);
    }

    function _bondPublish(uint256 inputRate, uint256 outputRate, uint256 computeRate) internal {
        vm.startPrank(provider);
        market.depositBond{ value: MINIMUM_BOND }();
        market.publishOffer(
            OFFER_ID, MODEL_HASH, CAPABILITY_HASH, inputRate, outputRate, computeRate
        );
        vm.stopPrank();
    }

    function _openSession(uint256 allowance, uint64 expiry) internal {
        vm.startPrank(customer);
        market.deposit{ value: allowance }();
        market.openSession(SESSION_ID, allowance, expiry);
        vm.stopPrank();
    }

    function _receipt(bytes32 requestId, uint64 nonce, uint256 maximumCharge)
        internal
        view
        returns (MyferenceMarket.Receipt memory)
    {
        return MyferenceMarket.Receipt({
            requestId: requestId,
            sessionId: SESSION_ID,
            customer: customer,
            provider: provider,
            settlementSigner: settlementSigner,
            offerId: OFFER_ID,
            priceVersion: 1,
            modelHash: MODEL_HASH,
            capabilityHash: CAPABILITY_HASH,
            inputTokens: 1_000_000,
            outputTokens: 0,
            computeMilliseconds: 0,
            maximumCharge: uint64(maximumCharge),
            totalCharge: 1 ether,
            feeBasisPoints: 500,
            feeVersion: 1,
            status: 1,
            completedAt: uint64(block.timestamp),
            inputHash: keccak256("prompt"),
            outputHash: keccak256("response"),
            nonce: nonce
        });
    }

    function _sign(uint256 privateKey, MyferenceMarket.Receipt memory receipt)
        internal
        view
        returns (bytes memory)
    {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, market.hashReceipt(receipt));
        return abi.encodePacked(r, s, v);
    }

    function _expectSettleRevert(
        bytes4 selector,
        MyferenceMarket.Receipt memory receipt,
        uint256 providerSigningKey,
        uint256 settlementSigningKey
    ) internal {
        bytes memory providerSignature = _sign(providerSigningKey, receipt);
        bytes memory settlementSignature = _sign(settlementSigningKey, receipt);
        vm.expectRevert(selector);
        market.settleReceipt(receipt, providerSignature, settlementSignature);
    }
}
