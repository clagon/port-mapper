export type FriendlyError = {
  title: string;
  message: string;
  details: string;
};

export function parseError(error: string): FriendlyError {
  if (!error) {
    return { title: 'エラー', message: '不明なエラーが発生しました。', details: '' };
  }

  const errStr = String(error).trim();

  // 1. ルーター未検出 / SSDP 探索失敗
  if (
    errStr.includes('no UPnP gateway discovered') ||
    errStr.includes('ErrNoGateway') ||
    errStr.includes('no fallback description responded') ||
    errStr.includes('no fallback control url responded')
  ) {
    return {
      title: 'ルーターが見つかりませんでした',
      message: 'お使いのルーターでUPnP（自動ポートマッピング機能）が有効になっているかご確認ください。また、PCがルーターと同じネットワーク（ルーターの配下）に接続されている必要があります。',
      details: errStr,
    };
  }

  // 2. ポート競合 (SOAP 718: ConflictInMappingEntry)
  if (
    errStr.includes('ConflictInMappingEntry') ||
    errStr.includes('718') ||
    errStr.toLowerCase().includes('conflict')
  ) {
    return {
      title: '指定されたポートはすでに使用されています',
      message: '設定しようとしたポート番号は、すでに他の機器や別のアプリケーションに使用されています。別のポート番号を指定してください。',
      details: errStr,
    };
  }

  // 3. ルーターの操作拒否 (SOAP 501 / 606等)
  if (errStr.includes('ActionFailed') || errStr.includes('501') || errStr.includes('606')) {
    return {
      title: 'ルーターが操作を拒否しました',
      message: 'ルーターのセキュリティ設定等により、ポート開放の要求がブロックされました。ルーターの管理者画面等でUPnP設定をご確認ください。',
      details: errStr,
    };
  }

  // 4. ネットワーク到達不可
  if (
    errStr.includes('network is unreachable') ||
    errStr.includes('connection refused') ||
    errStr.includes('route to host') ||
    errStr.includes('dial udp')
  ) {
    return {
      title: 'ネットワークに接続できません',
      message: 'ルーターへの接続が切断されているか、オフライン状態の可能性があります。Wi-FiやLANケーブルの接続状況をご確認ください。',
      details: errStr,
    };
  }

  // 5. 接続タイムアウト
  if (errStr.includes('timeout') || errStr.includes('Timeout') || errStr.includes('deadline exceeded')) {
    return {
      title: '通信がタイムアウトしました',
      message: 'ルーターからの応答が時間内にありませんでした。ルーターが非常に混雑しているか、再起動中の可能性があります。しばらくしてから再度お試しください。',
      details: errStr,
    };
  }

  // 6. 設定バリデーションエラー
  if (errStr.includes('invalid listen addr') || errStr.includes('must bind to localhost')) {
    return {
      title: '不正な設定値です',
      message: '入力された設定値が正しくありません。アドレスの形式（IP:ポート）が正しいか、および localhost 宛て（127.0.0.1）になっているかご確認ください。',
      details: errStr,
    };
  }

  // デフォルト（一般的なエラー）
  return {
    title: 'エラーが発生しました',
    message: '処理中に予期しない問題が発生しました。しばらく時間をおいてから再度お試しください。',
    details: errStr,
  };
}
