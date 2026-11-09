import 'package:flutter_test/flutter_test.dart';
import 'package:ssh_servers/sync.dart';

const key = 'AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8';

// sealed by sync.go (Windows app)
const goSealed = '{"app":"ssh-servers","v":1,"iv":"bnEctPrhUrXDBrC6",'
    '"ct":"jKDGP106g135NdQgNePyBIstLvj39I3cq9c2ffFdYi/iT57Q/vkAlqH6KMeTha/M+zOvoV1QvfsIL3iEMYLBKFQu4TQYqNqG5yM/dQ6SvpTORraR+GV9LbFdTtbtZ5OsC6QHhHxsm45v0mlnE7UyJKWtQtbuSRNhQdQwSbY+RR8yOQdq5gQ5BK5NfCOZEWrwqElQkMEu3xu8BU6dI9cg8wWYZSvJl1NL9MI="}';

Server s(String id, String name, int u) => Server(id: id, name: name, user: 'root', host: '$id.example', u: u);

void main() {
  test('opens a file sealed by the Windows app', () {
    final d = unseal(key, goSealed);
    expect(d.servers.single.name, 'Turkey');
    expect(d.servers.single.host, '1.2.3.4');
  });

  test('seal / unseal round trip', () {
    final d = SyncDoc(servers: [s('a', 'A', 5)], deleted: {'x': 1}, orderU: 3);
    final back = unseal(key, seal(key, d));
    expect(sameDoc(d, back), isTrue);
  });

  test('wrong key fails', () {
    expect(() => unseal('BBECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8', goSealed), throwsA(isA<SyncError>()));
  });

  test('stale device never brings old data back', () {
    final pc = SyncDoc(servers: [s('a', 'Old', 100), s('b', 'B', 50)]);
    final phone = SyncDoc(servers: [s('a', 'New', 200), s('c', 'C', 210)], deleted: {'b': 190});
    final m = mergeDocs(pc, phone);
    expect(m.servers.map((x) => x.name).toList(), ['New', 'C']);
    expect(sameDoc(m, mergeDocs(phone, pc)), isTrue);
  });

  test('sync link parsing', () {
    final sc = parseLink('https://corenetbase.com/SSH-Servers/app/#g=abc&k=$key&t=gho_x');
    expect(sc?.gist, 'abc');
    expect(parseLink('https://x/#g=abc&k=short&t=y'), isNull);
    expect(looksLikeToken('gho_${'a' * 36}'), isTrue);
  });
}
