/*
Build: gcc -D_REENTRANT -Wall -g -ggdb -o join join.c
*/

#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/ip.h>
#include <netinet/igmp.h>
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>
#include <net/if.h>
#include <arpa/inet.h>
#include <unistd.h>

const char *prog_name = "join";

static int join_group(int fd, int ifindex, struct in_addr group_addr)
{
  struct group_source_req req;
  struct sockaddr_in *group_sa = (struct sockaddr_in *) &req.gsr_group;
  //struct sockaddr_in *source_sa = (struct sockaddr_in *) &req.gsr_source;

  memset(group_sa, 0, sizeof(*group_sa));
  group_sa->sin_family = AF_INET;
  group_sa->sin_addr = group_addr;
  group_sa->sin_port = htons(0);

  /*
  memset(source_sa, 0, sizeof(*source_sa));
  source_sa->sin_family = AF_INET;
  source_sa->sin_addr = source_addr;
  source_sa->sin_port = htons(0);
  */

  req.gsr_interface = ifindex;

  //return setsockopt(fd, SOL_IP, MCAST_JOIN_SOURCE_GROUP, &req, sizeof(req));
  return setsockopt(fd, SOL_IP, MCAST_JOIN_GROUP, &req, sizeof(req));
}

static int name_to_index(const char *ifname)
{
  struct if_nameindex *ini;
  int ifindex = -1;
  int i;

  if (!ifname)
    return -1;

  ini = if_nameindex();
  if (!ini) {
    int err = errno;
    fprintf(stderr,
	    "%s: interface=%s: failure solving index: errno=%d: %s\n",
	    prog_name, ifname, err, strerror(err));
    errno = err;
    return -1;
  }

  for (i = 0; ini[i].if_index; ++i) {
    if (!strcmp(ini[i].if_name, ifname)) {
      ifindex = ini[i].if_index;
      break;
    }
  }

  if_freenameindex(ini);

  return ifindex;
}

static int index_to_name(int ifindex, char *buf, int buf_size)
{
  struct if_nameindex *ini;
  int i;
  int result = -1; // error

  if (!buf || buf_size < 2) {
    return -2; // ugh
  }

  ini = if_nameindex();
  if (!ini) {
    int err = errno;
    fprintf(stderr,
	    "%s: ifindex=%d: failure solving index: errno=%d: %s\n",
	    prog_name, ifindex, err, strerror(err));
    errno = err;
    return -3;
  }

  for (i = 0; ini[i].if_index; ++i) {
    if (ini[i].if_index == ifindex) {
      const char *ifname = ini[i].if_name;
      int len = strlen(ifname);
      if (len >= buf_size) {
	len = buf_size - 1;
	result = -4; // error
      } else {
	result = 0; // ok
      }
      memcpy(buf, ifname, len);
      buf[len] = '\0';
      break;
    }
  }

  if_freenameindex(ini);

  return result;
}

int recvfromto(int fd, uint8_t *buf, size_t len,
	       struct sockaddr_in *from, socklen_t *fromlen,
	       struct sockaddr_in *to, socklen_t *tolen,
	       int *ifindex)
{
  struct msghdr msgh;
  struct cmsghdr *cmsg;
  struct iovec iov;
  char cbuf[1000];
  int err;

  /*
   * IP_PKTINFO / IP_RECVDSTADDR don't yield sin_port.
   * Use getsockname() to get sin_port.
   */
  if (to) {
    struct sockaddr_in si;
    socklen_t si_len = sizeof(si);
    
    ((struct sockaddr_in *) to)->sin_family = AF_INET;

    if (getsockname(fd, (struct sockaddr *) &si, &si_len)) {
      ((struct sockaddr_in *) to)->sin_port        = ntohs(0);
      ((struct sockaddr_in *) to)->sin_addr.s_addr = ntohl(0);
    }
    else {
      ((struct sockaddr_in *) to)->sin_port = si.sin_port;
      ((struct sockaddr_in *) to)->sin_addr = si.sin_addr;
    }

    if (tolen) 
      *tolen = sizeof(si);
  }

  memset(&msgh, 0, sizeof(struct msghdr));
  iov.iov_base = buf;
  iov.iov_len  = len;
  msgh.msg_control = cbuf;
  msgh.msg_controllen = sizeof(cbuf);
  msgh.msg_name = from;
  msgh.msg_namelen = fromlen ? *fromlen : 0;
  msgh.msg_iov  = &iov;
  msgh.msg_iovlen = 1;
  msgh.msg_flags = 0;

  err = recvmsg(fd, &msgh, 0);
  if (err < 0)
    return err;

  if (fromlen)
    *fromlen = msgh.msg_namelen;

  for (cmsg = CMSG_FIRSTHDR(&msgh);
       cmsg != NULL;
       cmsg = CMSG_NXTHDR(&msgh,cmsg)) {

    if ((cmsg->cmsg_level == SOL_IP) && (cmsg->cmsg_type == IP_PKTINFO)) {
      struct in_pktinfo *i = (struct in_pktinfo *) CMSG_DATA(cmsg);
      if (to)
	((struct sockaddr_in *) to)->sin_addr = i->ipi_addr;
      if (tolen)
	*tolen = sizeof(struct sockaddr_in);
      if (ifindex)
	*ifindex = i->ipi_ifindex;
      break;
    }

  } /* for (cmsg) */

  return err; /* len */
}

static void read_loop(int fd) {

  struct sockaddr_in from;
  struct sockaddr_in to;
  socklen_t fromlen = sizeof(from);
  socklen_t tolen = sizeof(to);
  int ifindex = -1;
  uint8_t buf[10000];
  int rd;

  char str_from[100];
  char str_to[100];
  char ifname[100];

  for (;;) {
    rd = recvfromto(fd, buf, sizeof buf,
		    &from, &fromlen,
		    &to, &tolen,
		    &ifindex);
    if (rd < 0) {
      fprintf(stderr,
	      "%s: error: errno=%d: %s\n",
	      prog_name, errno, strerror(errno));
      continue;
    }

    inet_ntop(AF_INET, &from.sin_addr, str_from, sizeof str_from);
    inet_ntop(AF_INET, &to.sin_addr, str_to, sizeof str_to);
    index_to_name(ifindex, ifname, sizeof ifname);
    
    printf("%s: read %d bytes from %s:%d to %s:%d on %s ifindex=%d\n",
	   prog_name, rd, str_from, ntohs(from.sin_port), str_to, ntohs(to.sin_port),
	   ifname, ifindex);
  }
  
}

static void join(const char *ifname, const char *mcast, const char* group, const char *addr, const char *port_str) {
  int ifindex;
  struct in_addr iface_addr;
  struct in_addr group_addr;
  struct in_addr bind_addr;
  int port;
  int result;
  int fd;
  struct sockaddr_in sock_addr;
  
  ifindex = name_to_index(ifname);
  if (ifindex < 0) {
    fprintf(stderr, "%s: could not find interface: %s\n",
	    prog_name, ifname);
    exit(1);
  }

  result = inet_pton(AF_INET, mcast, &iface_addr);
  if (result <= 0) {
    fprintf(stderr, "%s: bad interface address: %s\n",
	    prog_name, mcast);
    exit(1);
  }
  
  result = inet_pton(AF_INET, group, &group_addr);
  if (result <= 0) {
    fprintf(stderr, "%s: bad group address: %s\n",
	    prog_name, group);
    exit(1);
  }
  
  result = inet_pton(AF_INET, addr, &bind_addr);
  if (result <= 0) {
    fprintf(stderr, "%s: bad bind address: %s\n",
	    prog_name, addr);
    exit(1);
  }

  port = atoi(port_str);

  fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_UDP);
  if (fd < 0) {
    fprintf(stderr,
	    "%s: could not create socket: socket(): errno=%d: %s\n",
	    prog_name, errno, strerror(errno));
    exit(1);
  }

  {
    int enable = 1; /* boolean */
    if (setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, (void *) &enable, sizeof enable)) {
      fprintf(stderr,
	      "%s: could not set SO_REUSEADDR: errno=%d: %s\n",
	      prog_name, errno, strerror(errno));
    }
  }

  {
    /* will request IP_PKTINFO info while reading the socket */
    int opt = 1;
    if (setsockopt(fd, SOL_IP, IP_PKTINFO, &opt, sizeof opt)) {
      fprintf(stderr, "%s: could not set IP_PKTINFO: errno=%d: %s\n",
	      prog_name, errno, strerror(errno));
    }
  }

  if (setsockopt(fd, IPPROTO_IP, IP_MULTICAST_IF, (void *) &iface_addr, sizeof iface_addr)) {
      fprintf(stderr, "%s: could not multicast interface: errno=%d: %s\n",
	      prog_name, errno, strerror(errno));
  }
  
  sock_addr.sin_family = AF_INET;
  sock_addr.sin_addr   = bind_addr;
  sock_addr.sin_port   = htons(port);

  if (bind(fd, (struct sockaddr *)&sock_addr, sizeof(sock_addr))) {
    fprintf(stderr,
	    "%s: could not bind socket: errno=%d: %s\n",
	    prog_name, errno, strerror(errno));
    exit(1);
  }
  
  if (join_group(fd, ifindex, group_addr)) {
    fprintf(stderr,
	    "%s: could not join group: errno=%d: %s\n",
	    prog_name, errno, strerror(errno));
    exit(1);
  }

  printf("%s: joined, reading...\n", prog_name);

  read_loop(fd);

  close(fd);
}

int main(int argc, const char *argv[]) {

  const char *ifname;
  const char *mcast;
  const char *group;
  const char *addr;
  const char *port;
  
  if (argc != 6) {
    fprintf(stderr,
            "usage:   %s interface multicast group     bind_addr port\n"
	    "example: %s eth2      1.0.0.2   224.0.0.9 0.0.0.0   2000\n",
	    prog_name, prog_name);
    exit(1);
  }

  ifname = argv[1];
  mcast = argv[2];
  group = argv[3];
  addr = argv[4];
  port = argv[5];

  join(ifname, mcast, group, addr, port);
  
  exit(0);
}
