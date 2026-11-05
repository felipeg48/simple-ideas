"""Authorise a card payment, retrying once when the bank times out."""
import time


def pay(bank, order, card, amount):
    # tag::retry[]
    key = order.id                                    # <1>
    result = bank.authorize(card, amount, key)
    if result.is_timeout:
        time.sleep(30)                                # <2>
        result = bank.authorize(card, amount, key)
    return result                                     # <3>
    # end::retry[]
