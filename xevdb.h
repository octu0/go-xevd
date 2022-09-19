#include <stdio.h>
#include <string.h>
#include "xevdb/xevd.h"

#ifndef H_GO_XEVDB
#define H_GO_XEVDB

typedef struct xevdb_decode_result_t {
  int status;
  int nalu_type;
  int slice_type;
  unsigned char *y;
  unsigned char *u;
  unsigned char *v;
  int stride_y;
  int stride_u;
  int stride_v;
  int size_y;
  int size_u;
  int size_v;
  int width;
  int height;
  int crop_top;
  int crop_right;
  int crop_bottom;
  int crop_left;
  int color_format;
  int bit_depth;
} xevdb_decode_result_t;

void xevdb_free_result(xevdb_decode_result_t *result) {
  if(result != NULL){
    free(result->y);
    free(result->u);
    free(result->v);
  }
  free(result);
}

void xevdb_free_xevd(XEVD id) {
  xevd_delete(id);
}

XEVD xevdb_create(int32_t threads) {
  XEVD_CDSC cdsc;
  cdsc.threads = threads;

  int ret;
  XEVD id = xevd_create(&cdsc, &ret);
  if(XEVD_FAILED(ret)) {
    return NULL;
  }
  return id;
}

void xevdb_free_imgb(XEVD_IMGB *imgb) {
  if(NULL != imgb) {
    int i;
    for(i = 0; i < XEVD_IMGB_MAX_PLANE; i += 1) {
      free(imgb->baddr[i]);
    }
  }
  free(imgb);
}

xevdb_decode_result_t *xevdb_decode(
  XEVD id,
  unsigned char *data,
  int32_t data_size
) {
  XEVD_BITB bitb;
  bitb.addr = data;
  bitb.ssize = data_size;

  XEVD_STAT stat;
  memset(&stat, 0, sizeof(XEVD_STAT));
  stat.read = XEVD_NAL_UNIT_LENGTH_BYTE; // data is no nalu_length header

  int ret_decode = xevd_decode(id, &bitb, &stat);
  if(XEVD_FAILED(ret_decode)) {
    return NULL;
  }

  xevdb_decode_result_t *result = (xevdb_decode_result_t *) malloc(sizeof(xevdb_decode_result_t));
  memset(result, 0, sizeof(xevdb_decode_result_t));
  result->y = NULL;
  result->u = NULL;
  result->v = NULL;

  if(stat.read < 1) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    return result;
  }

  XEVD_IMGB *imgb = (XEVD_IMGB *) malloc(sizeof(XEVD_IMGB));
  if(NULL == imgb) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    return result;
  }
  memset(imgb, 0, sizeof(XEVD_IMGB));

  int ret = xevd_pull(id, &imgb);
  if(XEVD_FAILED(ret)) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    xevdb_free_imgb(imgb);
    return result;
  }

  int bd = 1;
  if(10 <= XEVD_CS_GET_BIT_DEPTH(imgb->cs)) {
    bd = 2;
  }

  // stride includes width
  //result->size_y = imgb->h[0] * imgb->w[0];
  //result->size_u = imgb->h[1] * imgb->w[1];
  //result->size_v = imgb->h[2] * imgb->w[2];
  result->size_y = imgb->h[0] * imgb->s[0] * bd;
  result->size_u = imgb->h[1] * imgb->s[1] * bd;
  result->size_v = imgb->h[2] * imgb->s[2] * bd;
  result->stride_y = imgb->s[0] / bd;
  result->stride_u = imgb->s[1] / bd;
  result->stride_v = imgb->s[2] / bd;
  result->width = imgb->w[0];
  result->height = imgb->h[0];
  result->crop_top = imgb->crop_t;
  result->crop_right = imgb->crop_r;
  result->crop_bottom = imgb->crop_b;
  result->crop_left = imgb->crop_l;
  result->color_format = XEVD_CS_GET_FORMAT(imgb->cs);
  result->bit_depth = XEVD_CS_GET_BIT_DEPTH(imgb->cs);

  result->y = (unsigned char*) malloc(result->size_y);
  if(NULL == result->y) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    xevdb_free_imgb(imgb);
    return result;
  }
  result->u = (unsigned char*) malloc(result->size_u);
  if(NULL == result->u) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    xevdb_free_imgb(imgb);
    return result;
  }
  result->v = (unsigned char*) malloc(result->size_v);
  if(NULL == result->v) {
    result->status = XEVD_OK_OUT_NOT_AVAILABLE;
    xevdb_free_imgb(imgb);
    return result;
  }
  memcpy(result->y, imgb->a[0], result->size_y);
  memcpy(result->u, imgb->a[1], result->size_u);
  memcpy(result->v, imgb->a[2], result->size_v);
  result->nalu_type = stat.nalu_type;
  result->slice_type = stat.stype;
  result->status = ret;

  xevdb_free_imgb(imgb);
  return result;
}

#endif
