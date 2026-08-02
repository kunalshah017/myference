// SPDX-License-Identifier: MIT
pragma solidity ^0.8.30;

import { Test } from "forge-std/Test.sol";
import { MyferenceMarket } from "../src/MyferenceMarket.sol";

contract MyferenceMarketTest is Test {
    MyferenceMarket internal market;

    address internal owner = makeAddr("owner");
    address internal customer = makeAddr("customer");
    address internal provider = makeAddr("provider");
    address internal feeRecipient = makeAddr("feeRecipient");
    address internal settlementSigner = makeAddr("settlementSigner");

    uint256 internal constant MINIMUM_BOND = 10 ether;
    uint64 internal constant EXIT_DELAY = 7 days;
    bytes32 internal constant OFFER_ID = keccak256("ollama/qwen3:8b");
    bytes32 internal constant MODEL_HASH = keccak256("qwen3:8b");
    bytes32 internal constant CAPABILITY_HASH = keccak256("chat-completions");

    function setUp() public {
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
}
