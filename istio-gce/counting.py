#! /usr/bin/python

from lxml import html
import requests


def send_request():
  page = requests.get('http://35.226.99.75/')
  tree = html.fromstring(page.content)
  elements = tree.xpath('/html/body/main/div/div/div/div/div/div/h5')
  return [e.text.strip() for e in elements]

vm = 0
kubernetes = 0
for i in xrange(3000000):
  products = send_request()
  if len(products) == 1:
    vm += 1
  else:
    kubernetes += 1
  print "Kubernetes vs VM requests handled %d / %d, products: %s" % (kubernetes, vm, products)
