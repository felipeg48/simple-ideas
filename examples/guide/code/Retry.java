package shop.payments;

import java.time.Duration;

/** Authorises a card payment, retrying once when the bank times out. */
public final class Retry {
    private final Bank bank;

    public Retry(Bank bank) { this.bank = bank; }

    public PaymentResult pay(Order order, Card card, Money amount) throws InterruptedException {
        // tag::retry[]
        var key = order.id();                                  // <1>
        var result = bank.authorize(card, amount, key);
        if (result.isTimeout()) {
            Thread.sleep(Duration.ofSeconds(30));              // <2>
            result = bank.authorize(card, amount, key);
        }
        return result;                                         // <3>
        // end::retry[]
    }
}
