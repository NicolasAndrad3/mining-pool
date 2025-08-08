// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/security/Pausable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";

contract PoolPayoutManager is ReentrancyGuard, Pausable, Ownable {
    address public immutable token;
    address public validator;

    event ValidatorUpdated(address indexed previousValidator, address indexed newValidator);
    event PayoutExecuted(address indexed to, uint256 amount, uint256 timestamp, uint256 blockNumber);
    event FundsRecovered(address indexed recipient, uint256 amount);

    modifier onlyValidator() {
        require(msg.sender == validator, "Unauthorized: caller is not validator");
        _;
    }

    constructor(address _tokenAddress, address _initialValidator) {
        require(_tokenAddress != address(0), "Invalid token address");
        require(_initialValidator != address(0), "Invalid validator address");

        token = _tokenAddress;
        validator = _initialValidator;
        _transferOwnership(msg.sender);
    }

    function updateValidator(address _newValidator) external onlyOwner {
        require(_newValidator != address(0), "Validator cannot be zero address");
        emit ValidatorUpdated(validator, _newValidator);
        validator = _newValidator;
    }

    function payMiner(address _to, uint256 _amount) external nonReentrant whenNotPaused onlyValidator {
        require(_to != address(0), "Invalid recipient");
        require(_amount > 0, "Amount must be greater than zero");

        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance >= _amount, "Insufficient contract balance");

        bool success = IERC20(token).transfer(_to, _amount);
        require(success, "Token transfer failed");

        emit PayoutExecuted(_to, _amount, block.timestamp, block.number);
    }

    function recoverFunds(address _recipient) external onlyOwner whenPaused {
        require(_recipient != address(0), "Invalid recipient address");

        uint256 balance = IERC20(token).balanceOf(address(this));
        require(balance > 0, "No funds to recover");

        bool transferred = IERC20(token).transfer(_recipient, balance);
        require(transferred, "Fund recovery failed");

        emit FundsRecovered(_recipient, balance);
    }

    function pause() external onlyOwner {
        _pause();
    }

    function unpause() external onlyOwner {
        _unpause();
    }
}
