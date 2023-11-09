import {
  OrderTable,
  OrderStatus,
} from '@dydxprotocol-indexer/postgres';
import { getOrderIdHash } from '@dydxprotocol-indexer/v4-proto-parser';
import {
  OffChainUpdateV1,
  IndexerOrderId,
  OrderRemoveV1_OrderRemovalStatus,
  StatefulOrderEventV1,
} from '@dydxprotocol-indexer/v4-protos';

import config from '../../config';
import { ConsolidatedKafkaEvent } from '../../lib/types';
import { AbstractStatefulOrderHandler } from '../abstract-stateful-order-handler';
import * as pg from "pg";

export class StatefulOrderRemovalHandler extends
  AbstractStatefulOrderHandler<StatefulOrderEventV1> {
  eventType: string = 'StatefulOrderEvent';

  public getParallelizationIds(): string[] {
    // Stateful Order Events with the same orderId
    const orderId: string = OrderTable.orderIdToUuid(this.event.orderRemoval!.removedOrderId!);
    return this.getParallelizationIdsFromOrderId(orderId);
  }

  // eslint-disable-next-line @typescript-eslint/require-await
  public async internalHandle(resultRow: pg.QueryResultRow | undefined): Promise<ConsolidatedKafkaEvent[]> {
    if (config.USE_STATEFUL_ORDER_HANDLER_SQL_FUNCTION) {
      return this.handleViaSqlFunction(resultRow);
    }
    return this.handleViaKnex();
  }

  private async handleViaSqlFunction(resultRow: pg.QueryResultRow | undefined): Promise<ConsolidatedKafkaEvent[]> {
    const orderIdProto: IndexerOrderId = this.event.orderRemoval!.removedOrderId!;
    await this.handleEventViaSqlFunction(resultRow);
    return this.createKafkaEvents(orderIdProto);
  }

  private async handleViaKnex(): Promise<ConsolidatedKafkaEvent[]> {
    const orderIdProto: IndexerOrderId = this.event.orderRemoval!.removedOrderId!;
    await this.runFuncWithTimingStatAndErrorLogging(
      this.updateOrderStatus(orderIdProto, OrderStatus.CANCELED),
      this.generateTimingStatsOptions('cancel_order'),
    );

    return this.createKafkaEvents(orderIdProto);
  }

  private createKafkaEvents(orderIdProto: IndexerOrderId): ConsolidatedKafkaEvent[] {
    const offChainUpdate: OffChainUpdateV1 = OffChainUpdateV1.fromPartial({
      orderRemove: {
        removedOrderId: orderIdProto,
        reason: this.event.orderRemoval!.reason,
        removalStatus: OrderRemoveV1_OrderRemovalStatus.ORDER_REMOVAL_STATUS_CANCELED,
      },
    });

    return [
      this.generateConsolidatedVulcanKafkaEvent(
        getOrderIdHash(orderIdProto),
        offChainUpdate,
      ),
    ];
  }
}
