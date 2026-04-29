package kafka

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

type ReplayOptions struct {
	MaxMessages int
	Timeout     time.Duration
}

func DefaultReplayOptions() ReplayOptions {
	return ReplayOptions{
		MaxMessages: 0, // read all available
		Timeout:     30 * time.Second,
	}
}

// ReplayTopic reads all available messages from a topic.
/* func (eb *EventBus) ReplayTopic(ctx context.Context, topic string, opts ReplayOptions) ([]kafka.Message, error) {
	eb.logger.Info("Replaying topic", "topic", topic, "max_messages", opts.MaxMessages, "timeout", opts.Timeout)

	replayReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  eb.kafkaCfg.Brokers,
		Topic:    topic,
		GroupID:  "replay-" + topic + "-" + eb.kafkaCfg.GroupID,
		MinBytes: 1,
		MaxBytes: 10 * 1024 * 1024, // 10MB batches for throughput
	})
	defer replayReader.Close()

	var messages []kafka.Message
	deadline := time.After(opts.Timeout)

	for {
		select {
		case <-ctx.Done():
			return messages, ctx.Err()
		case <-deadline:
			eb.logger.Warn("Replay timeout reached", "topic", topic, "messages_read", len(messages))
			return messages, nil // return what we have, don't fail
		default:
			m, err := replayReader.FetchMessage(ctx)
			if err != nil {
				if err == io.EOF || err.Error() == "kafka: no message on queue" {
					eb.logger.Info("Replay complete: no more messages on topic", "topic", topic, "messages_read", len(messages))
					return messages, nil
				}
				eb.logger.Debug("Replay fetch ended with error", "topic", topic, "error", err, "messages_read", len(messages))
				return messages, nil
			}
			messages = append(messages, m)
			if opts.MaxMessages > 0 && len(messages) >= opts.MaxMessages {
				eb.logger.Info("Replay max messages reached", "topic", topic, "messages_read", len(messages))
				return messages, nil
			}
		}
	}
}
*/
// ReplayTopic reads all available messages from a topic up to the
// high-water mark at the time of the call, so it never blocks on an
// empty (or fully-consumed) topic.
func (eb *EventBus) ReplayTopic(ctx context.Context, topic string, opts ReplayOptions) ([]kafka.Message, error) {
	eb.logger.Info("Replaying topic", "topic", topic, "max_messages", opts.MaxMessages, "timeout", opts.Timeout)

	replayCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	var messages []kafka.Message

	// 1. Discover partitions and their high-water marks (HWM).
	//    The HWM is the offset of the *next* message to be written,
	//    so any offset < HWM is a message that already exists.
	conn, err := kafka.DialLeader(replayCtx, "tcp", eb.kafkaCfg.Brokers[0], topic, 0)
	if err != nil {
		eb.logger.Debug("Replay connection failed with error", "topic", topic, "dial leader", err)
		return messages, nil
	}

	partitions, err := conn.ReadPartitions(topic)
	conn.Close()
	if err != nil {
		eb.logger.Debug("Replay connection failed with error", "topic", topic, "read partitions", err)
		return messages, nil
	}

	type partitionBound struct {
		partition   int
		firstOffset int64
		hwm         int64
	}
	var bounds []partitionBound
	for _, p := range partitions {
		pc, err := kafka.DialLeader(replayCtx, "tcp", p.Leader.Host+":"+strconv.Itoa(p.Leader.Port), topic, p.ID)
		if err != nil {
			eb.logger.Debug("Replay connection failed with error", "topic", topic, "dial partition", p.ID, "error", err)
			return messages, nil
		}

		first, err := pc.ReadFirstOffset()
		if err != nil {
			pc.Close()
			eb.logger.Debug("Replay connection failed with error", "topic", topic, "read first offset partition", p.ID, "error", err)
			return messages, nil
		}

		hwm, err := pc.ReadLastOffset()
		pc.Close()
		if err != nil {
			eb.logger.Debug("Replay connection failed with error", "topic", topic, "read last offset partition", p.ID, "error", err)
			return messages, nil
		}

		// first == hwm means the partition is empty (nothing to consume)
		if hwm > first {
			bounds = append(bounds, partitionBound{p.ID, first, hwm})
		}
	}

	// 2. Topic is completely empty — nothing to replay.
	if len(bounds) == 0 {
		eb.logger.Info("Replay complete: topic is empty", "topic", topic)
		return messages, nil
	}

	// 3. Read each partition from the beginning up to its HWM.
	for _, b := range bounds {
		if opts.MaxMessages > 0 && len(messages) >= opts.MaxMessages {
			break
		}
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:   eb.kafkaCfg.Brokers,
			Topic:     topic,
			Partition: b.partition,
			MinBytes:  1,
			MaxBytes:  10 * 1024 * 1024,
		})
		// Start from the very beginning of the partition.
		if err := r.SetOffset(b.firstOffset); err != nil {
			r.Close()
			eb.logger.Debug("Replay connection failed with error", "topic", topic, "set offset partition", b.partition, "error", err)
			return messages, nil
		}

		for {
			if opts.MaxMessages > 0 && len(messages) >= opts.MaxMessages {
				break
			}
			m, err := r.FetchMessage(replayCtx)
			if err != nil {
				// Context deadline = our timeout; treat as "done".
				if errors.Is(err, context.DeadlineExceeded) {
					eb.logger.Warn("Replay timeout reached", "topic", topic, "messages_read", len(messages))
					r.Close()
					return messages, nil
				}
				r.Close()
				eb.logger.Debug("Replay fetch ended with error", "topic", topic, "partition", b.partition, "error", err, "messages_read", len(messages))
				return messages, nil
			}
			messages = append(messages, m)

			// Stop once we've reached the HWM captured before we started.
			// Offset is 0-based, so offset == hwm-1 is the last existing message.
			if m.Offset >= b.hwm-1 {
				break
			}
		}
		r.Close()
	}

	eb.logger.Info("Replay complete", "topic", topic, "messages_read", len(messages))
	return messages, nil
}
